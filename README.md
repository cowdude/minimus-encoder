# minimus-encoder


## Project description

Data stream encoder/decoder for compressing sequences of float/integer values to bits.

It focuses on encoding vectors of elements, i.e. interleaved series.

Heavily inspired by [Facebook's Gorilla TSDB paper](http://www.vldb.org/pvldb/vol8/p1816-teller.pdf).

Shipped with a lossy float64 transform function, allowing more efficient (but lossy) storage. This module can still be used for lossless encoding.

Thanks [icza](https://github.com/icza) for his `bitio` library, heavily used in this project.


## Current status

The project is quite young and could definitely use more testing and eventually some optimizations. I am currently using it to store large amounts of fixed-interval time-series on the cloud.

There is currently an issue with values that rapidly oscillates around zero (i.e. flipping the float64's sign bit too often). In case you would like to use this project, it is best advised to apply a bias to your inputs in order to avoid flipping the sign bit too often.

Pull requests and suggestions are welcome, feel free to open an issue.


## Example

See the `example` directory. The program compresses/uncompresses an arbitrary
sequence of vectors with different loss levels. Finally, it prints out both
inputs and outputs sequences, and displays the mean compressed data bits
per sample.

```
go run ./example

        0: 60.4700  0.0163  1.0000  1.0000  => 60.3750  0.0156  1.0000  1.0000
        1: 94.0500  0.0105  1.0000  1.0000  => 94.0000  0.0078  1.0000  1.0000
        2: 66.4600  0.0148  1.0000  1.0000  => 66.3750  0.0117  1.0000  1.0000
      999: 45.0700  0.0217 32.0000 77.0000  => 45.0000  0.0156 32.0000 77.0000
     1000:  1.1111  2.2222  3.3333  4.4444  =>  1.0625  2.1250  3.2500  4.3750
|e|=1e-01:  6.999 b/sample

        0: 60.4700  0.0163  1.0000  1.0000  => 60.4700  0.0162  1.0000  1.0000
        1: 94.0500  0.0105  1.0000  1.0000  => 94.0499  0.0105  1.0000  1.0000
        2: 66.4600  0.0148  1.0000  1.0000  => 66.4600  0.0148  1.0000  1.0000
      999: 45.0700  0.0217 32.0000 77.0000  => 45.0699  0.0216 32.0000 77.0000
     1000:  1.1111  2.2222  3.3333  4.4444  =>  1.1111  2.2222  3.3333  4.4444
|e|=1e-04: 11.838 b/sample

        0: 60.4700  0.0163  1.0000  1.0000  => 60.4700  0.0163  1.0000  1.0000
        1: 94.0500  0.0105  1.0000  1.0000  => 94.0500  0.0105  1.0000  1.0000
        2: 66.4600  0.0148  1.0000  1.0000  => 66.4600  0.0148  1.0000  1.0000
      999: 45.0700  0.0217 32.0000 77.0000  => 45.0700  0.0217 32.0000 77.0000
     1000:  1.1111  2.2222  3.3333  4.4444  =>  1.1111  2.2222  3.3333  4.4444
|e|=1e-07: 16.819 b/sample

        0: 60.4700  0.0163  1.0000  1.0000  => 60.4700  0.0163  1.0000  1.0000
        1: 94.0500  0.0105  1.0000  1.0000  => 94.0500  0.0105  1.0000  1.0000
        2: 66.4600  0.0148  1.0000  1.0000  => 66.4600  0.0148  1.0000  1.0000
      999: 45.0700  0.0217 32.0000 77.0000  => 45.0700  0.0217 32.0000 77.0000
     1000:  1.1111  2.2222  3.3333  4.4444  =>  1.1111  2.2222  3.3333  4.4444
|e|=1e-10: 21.762 b/sample

        0: 60.4700  0.0163  1.0000  1.0000  => 60.4700  0.0163  1.0000  1.0000
        1: 94.0500  0.0105  1.0000  1.0000  => 94.0500  0.0105  1.0000  1.0000
        2: 66.4600  0.0148  1.0000  1.0000  => 66.4600  0.0148  1.0000  1.0000
      999: 45.0700  0.0217 32.0000 77.0000  => 45.0700  0.0217 32.0000 77.0000
     1000:  1.1111  2.2222  3.3333  4.4444  =>  1.1111  2.2222  3.3333  4.4444
|e|=1e-13: 26.543 b/sample

        0: 60.4700  0.0163  1.0000  1.0000  => 60.4700  0.0163  1.0000  1.0000
        1: 94.0500  0.0105  1.0000  1.0000  => 94.0500  0.0105  1.0000  1.0000
        2: 66.4600  0.0148  1.0000  1.0000  => 66.4600  0.0148  1.0000  1.0000
      999: 45.0700  0.0217 32.0000 77.0000  => 45.0700  0.0217 32.0000 77.0000
     1000:  1.1111  2.2222  3.3333  4.4444  =>  1.1111  2.2222  3.3333  4.4444
|e|=1e-16: 29.764 b/sample

        0: 60.4700  0.0163  1.0000  1.0000  => 60.4700  0.0163  1.0000  1.0000
        1: 94.0500  0.0105  1.0000  1.0000  => 94.0500  0.0105  1.0000  1.0000
        2: 66.4600  0.0148  1.0000  1.0000  => 66.4600  0.0148  1.0000  1.0000
      999: 45.0700  0.0217 32.0000 77.0000  => 45.0700  0.0217 32.0000 77.0000
     1000:  1.1111  2.2222  3.3333  4.4444  =>  1.1111  2.2222  3.3333  4.4444
(lossless)    30.078 b/sample
```