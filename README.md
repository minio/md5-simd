
# md5-simd

SIMD accelerated MD5 package, allow up to 8 independent MD5 sums to be calculated on a single core.

This package is based upon the [md5vec](https://github.com/igneous-systems/md5vec) repository by Igneous Systems. It integrates a similar mechanism as described in https://github.com/minio/sha256-simd#support-for-avx512 for making it easy for clients to take advantages of the parallel nature of the MD5 calculation, resulting in reduced overall CPU load. 

## Performance

### block function
AVX2 (= 8 lanes) vs AVX512 (= 16 lanes) `block()` performance:

```
BenchmarkBlock8-4        9695575               124 ns/op        4144.80 MB/s           0 B/op          0 allocs/op
BenchmarkBlock16-4       7173894               167 ns/op        6122.07 MB/s           0 B/op          0 allocs/op
```

### hash.Hash

`crypto/md5` vs AVX2

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

`crypto/md5` vs AVX512

```
benchmark                   old MB/s     new MB/s     speedup
BenchmarkGolden/32KB-4      688.29       3427.50      4.98x
BenchmarkGolden/64KB-4      687.97       3788.35      5.51x
BenchmarkGolden/128KB-4     687.91       3612.76      5.25x
BenchmarkGolden/256KB-4     687.84       3800.89      5.53x
BenchmarkGolden/512KB-4     687.94       3832.28      5.57x
BenchmarkGolden/1MB-4       687.88       4086.52      5.94x
BenchmarkGolden/2MB-4       687.75       3295.48      4.79x
```

AVX2 vs AVX512

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