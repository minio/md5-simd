
# md5-simd

SIMD accelerated MD5 package, allow up to 8 independent MD5 sums to be calculated on a single core.

This package is based upon the [md5vec](https://github.com/igneous-systems/md5vec) repository by Igneous Systems. It integrates a similar mechanism as described in https://github.com/minio/sha256-simd#support-for-avx512 for making it easy for clients to take advantages of the parallel nature of the MD5 calculation, resulting in reduced overall CPU load. 

## Performance

Single core performance (aggregated) for different block sizes:

```
BenchmarkGolden/32KB-8             14199             85032 ns/op        3082.88 MB/s        1712 B/op         24 allocs/op
BenchmarkGolden/64KB-8              7680            156312 ns/op        3354.11 MB/s        1838 B/op         26 allocs/op
BenchmarkGolden/128KB-8             3788            303343 ns/op        3456.74 MB/s        2148 B/op         30 allocs/op
BenchmarkGolden/256KB-8             1954            612922 ns/op        3421.57 MB/s        2921 B/op         42 allocs/op
BenchmarkGolden/512KB-8              860           1383787 ns/op        3031.03 MB/s        5092 B/op         73 allocs/op
BenchmarkGolden/1MB-8                408           2904512 ns/op        2888.13 MB/s       10242 B/op        147 allocs/op
BenchmarkGolden/2MB-8                202           5908945 ns/op        2839.29 MB/s       22042 B/op        317 allocs/op
```

### TODO
 
- [ ] Add support for varying messages lengths
- [ ] Investigate support for AVX512 (offering up to 16x parallel execution)
