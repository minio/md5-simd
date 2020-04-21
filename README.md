
# md5-simd

This is a SIMD accelerated MD5 package, allowing up to either 8 (AVX2) or 16 (AVX512) independent MD5 sums to be calculated on a single CPU core.

It was originally based on the [md5vec](https://github.com/igneous-systems/md5vec) repository by Igneous Systems, but has been made more flexible by  amongst others supporting different message sizes per lane.

`md5-simd` integrates a similar mechanism as described in https://github.com/minio/sha256-simd#support-for-avx512 for making it easy for clients to take advantages of the parallel nature of the MD5 calculation. This will result in reduced overall CPU load. 

It is important to understand that `md5-simd` **does not speed up** an individual MD5 hash sum (unless you would be using some hierarchical tree structure). Rather it allows multiple __independent__  MD5 sums to be computed in parallel on the same CPU core, thereby making more efficient usage of the computing resources.

## Usage

In order to use `md5-simd`, you must first create an `Md5Server` which can subsequently be used to instantiate one (or more) objects for MD5 hashing. These objects conform to the regular `hash.Hash` interface and as such the normal Write/Reset/Sum functionality works as expected. 

As an example: 
```
    // Create server
    server := md5simd.NewServer()

    // Create hashing object (conforming to hash.Hash)
    md5Hash := server.NewHash()
    
    // Write one (or more) blocks
    md5Hash.Write(block)
    
    // Return digest
    digest := md5Hash.Sum([]byte{})
```

## Performance

The following chart compares the single-core performance between `crypto/md5` vs the AVX2 vs the AVX512 code:

![md5-performance-overview](chart/Single-core-MD5-Aggregated-Hashing-Performance.png)

Compared to `crypto/md5`, the AVX2 version is about 2.5 to 3.5 times faster:

```
benchmark                   old MB/s     new MB/s     speedup
BenchmarkGolden/32KB-4      688.29       2928.75      4.26x
BenchmarkGolden/64KB-4      687.97       2937.95      4.27x
BenchmarkGolden/128KB-4     687.91       2676.93      3.89x
BenchmarkGolden/256KB-4     687.84       2644.37      3.84x
BenchmarkGolden/512KB-4     687.94       2630.64      3.82x
BenchmarkGolden/1MB-4       687.88       2030.45      2.95x
BenchmarkGolden/2MB-4       687.75       1732.51      2.52x
```

Compared to AVX2, the AVX512 is up to 2x faster (especially for larger blocks)

```
benchmark                   old MB/s     new MB/s     speedup
BenchmarkGolden/32KB-4      2928.75      3427.50      1.17x
BenchmarkGolden/64KB-4      2937.95      3788.35      1.29x
BenchmarkGolden/128KB-4     2676.93      3612.76      1.35x
BenchmarkGolden/256KB-4     2644.37      3800.89      1.44x
BenchmarkGolden/512KB-4     2630.64      3832.28      1.46x
BenchmarkGolden/1MB-4       2030.45      4086.52      2.01x
BenchmarkGolden/2MB-4       1732.51      3295.48      1.90x
```

These measurements were performed on AWS EC2 instance of type c5.xlarge equipped with a Xeon Platinum 8124M CPU at 3.0 GHz.

## Design

md5-simd has both an AVX2 (8-lane parallel) and an AVX512 (16-lane parallel version) algorithm to accelerate the computation with the following function definitions:
```
//go:noescape
func block8(state *uint32, base uintptr, bufs *int32, cache *byte, n int)

//go:noescape
func block16(state *uint32, ptrs *int64, mask uint64, n int)
```

The AVX2 version is based on the [md5vec](https://github.com/igneous-systems/md5vec) repository and is essentially unchanged except for minor (cosmetic) changes.

The AVX512 version is derived from the AVX2 version but adds some further optimizations and simplifications.

### Caching in upper ZMM registers

The AVX2 version passes in a `cache8` block of memory (about 0.5 KB) for temporary storage of intermediate results during `ROUND1` which are subsequently used during `ROUND2` through to `ROUND4`.

Since AVX512 has double the amount of registers (32 ZMM registers as compared to 16 YMM registers), it is possible to use the upper 16 ZMM registers for keeping the intermediate states on the CPU. As such, there is no need to pass in a corresponding `cache16` into the AVX512 block function.

### Direct loading using 64-bit pointers

The AVX2 uses the `VPGATHERDD` instruction (for YMM) to do a parallel load of 8 lanes using (8 independent) 32-bit offets. Since there is no control over how the 8 slices that are passed into the (Golang) `blockMd5` function are laid out into memory, it is not possible to derive a "base" address and corresponding offsets (all within 32-bits) for all 8 slices.

As such the AVX2 version uses an interim buffer to collect the byte slices to be hashed from all 8 inut slices and passed this buffer along with (fixed) 32-bit offsets into the assembly code.

For the AVX512 version this interim buffer is not needed since the AVX512 code uses a pair of `VPGATHERQD` instructions to directly dereference 64-bit pointers (from a base register address that is initialized to zero).

Note that two load (gather) instructions are needed because the AVX512 version processes 16-lanes in parallel, requiring 16 times 64-bit = 1024 bits in total to be loaded. A simple `VALIGND` and `VPORD` are subsequently used to merge the lower and upper halves together into a single ZMM register (that contains 16 lanes of 32-bit DWORDS).

### Masking support

Due to the fact that pointers are directly passed in from the Golang slices, we need to protect against NULL pointers. For this a 16-bit mask is passed in the AVX512 assembly code which is used during the `VPGATHERQD` instructions to mask out lanes that could otherwise result in segment violations.

## Low level block function performance

The benchmark below shows the (single thread) maximum performance of the `block()` function for AVX2 (having 8 lanes) and AVX512 (having 16 lanes) performance:

```
BenchmarkBlock8-4        9695575               124 ns/op        4144.80 MB/s           0 B/op          0 allocs/op
BenchmarkBlock16-4       7173894               167 ns/op        6122.07 MB/s           0 B/op          0 allocs/op
```

## Limitations

As explained above `md5-simd` does not speed up an individual MD5 hash sum computation (unless some hierarchical tree construct is used but this will result in different outcomes).

Instead it allows to run multiple MD5 calculations in parallel on a single CPU core. This can be beneficial in e.g. multi-threaded server applications where many go-routines are dealing with many requests and multiple MD5 calculations can be packed/scheduled for parallel execution on a single core.

This will result in a lower overall CPU usage as compared to using the standard `crypto/md5` functionality where each MD5 hash computation will consume a single thread (core).

It is best to test and measure the overall CPU usage in a representative usage scenario in your application to get an overall understanding of the benefits of `md5-simd` as compared to `crypto/md5` (ideally under heavy CPU load).

Also note that `md5-simd` is best meant to work with large objects, so if your application only hashes small objects (KB-size rather than MB-size), you may be better of by using `crypto/md5`.

## License

`md5-simd` is released under the MIT License. You can find the complete text in the file LICENSE.

## Contributing

Contributions are welcome, please send PRs for any enhancements.