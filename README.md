
# md5-simd

SIMD accelerated MD5 package, allow up to 8 independent MD5 sums to be calculated on a single core.

This package is based upon the [md5vec](https://github.com/igneous-systems/md5vec) repository by Igneous Systems. It integrates a similar mechanism as described in https://github.com/minio/sha256-simd#support-for-avx512 for making it easy for clients to take advantages of the parallel nature of the MD5 calculation, resulting in reduced overall CPU load. 

## Performance

Single core performance (aggregated) for different block sizes:

```
BenchmarkGolden/32KB-8             26373             44667 ns/op        5868.83 MB/s      264413 B/op         33 allocs/op
BenchmarkGolden/64KB-8             15537             77450 ns/op        6769.42 MB/s      526638 B/op         34 allocs/op
BenchmarkGolden/128KB-8             7994            146630 ns/op        7151.19 MB/s     1051115 B/op         37 allocs/op
BenchmarkGolden/256KB-8             3684            279915 ns/op        7492.10 MB/s     2100121 B/op         43 allocs/op
BenchmarkGolden/512KB-8             1816            616046 ns/op        6808.42 MB/s     4198477 B/op         60 allocs/op
BenchmarkGolden/1MB-8                837           1356011 ns/op        6186.24 MB/s     8395747 B/op        103 allocs/op
BenchmarkGolden/2MB-8                450           2530872 ns/op        6629.03 MB/s    16789713 B/op        180 allocs/op
BenchmarkGolden/5MB-8                170           6599136 ns/op        6355.84 MB/s    41974274 B/op        449 allocs/op
```

### TODO
 
- [ ] Add support for varying messages lengths
- [ ] Investigate support for AVX512 (offering up to 16x parallel execution)
