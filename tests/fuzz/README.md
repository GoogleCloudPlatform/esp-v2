## Fuzz Tests

Fuzz tests are defined as `*_fuzz_test.cc` files in [src/envoy/](../../src/envoy) directories.

The [corpus/](./corpus) directory contains input data for each fuzz test.

### Regression Tests

When running presubmits, the fuzz tests are run **without** a fuzzing engine.
Only the corpus data will be used for each fuzz test.
Therefore, the corpus should contain at least a few files that are known to cause bugs.
This serves as regression tests.

### Discovery Tests

When running continuously, the fuzz tests are run with a fuzzing engine to discover new bugs.
We intend the tests to be run with [LibFuzzer](https://llvm.org/docs/LibFuzzer.html), but are compatible with other engines.

LibFuzzer is a coverage-guided, evolutionary fuzzing engine.
Therefore, the corpus should contain a few inputs that LibFuzzer can modify.
When a bug is discovered via continuous fuzzing, the input data should be added to the corresponding test's corpus to serve as a regression test in presubmits.