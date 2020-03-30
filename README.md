
# md5-simd

SIMD accelerated MD5 package, allow up to 8 independent MD5 sums to be calculated on a single core.

This package is based upon the [md5vec](https://github.com/igneous-systems/md5vec) repository by Igneous Systems. It integrates a similar mechanism as described in https://github.com/minio/sha256-simd#support-for-avx512 for making it easy for clients to take advantages of the parallel nature of the MD5 calculation, resulting in reduced overall CPU load. 

```
BenchmarkMd5-8               100          11492754 ns/op         729.90 MB/s
BenchmarkMd5by8-8            573           2159327 ns/op        3884.83 MB/s
```

### TODO
 
- [ ] Add support for varying messages lengths
- [ ] Investigate support for AVX512 (offering up to 16x parallel execution)
