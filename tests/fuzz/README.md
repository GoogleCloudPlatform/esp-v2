## Fuzz Tests

This document explains how fuzz tests are structured and run in this repo.

### Repository Structure

- Fuzz tests are defined as `*_fuzz_test.cc` files in [src/envoy/](../../src/envoy) directories.
- The [corpus/](./corpus) directory contains seed input data for each fuzz test.
This seed corpus serves as regression tests and increases continuous fuzzing efficiency.
See [Good Fuzz Target: Seed Corpus](https://github.com/google/fuzzing/blob/master/docs/structure-aware-fuzzing.md#example-protocol-buffers)
for more details.
- The [structured_inputs/](./structured_inputs) directory contains protos that serve
as the structured inputs for fuzz tests. Structure-aware fuzz tests can use these protos
as the input format (instead of just receiving a buffer as an input). See
[Structure-Aware Fuzzing: Protocol Buffers](https://github.com/google/fuzzing/blob/master/docs/structure-aware-fuzzing.md#example-protocol-buffers)
for more details.

### Running the Fuzz Tests

This section gives examples of how to run the fuzz tests locally using [LibFuzzer](https://llvm.org/docs/LibFuzzer.html).

#### Prerequisites

You need LLVM and `clang-8` to run fuzz tests with LibFuzzer.

```.shell script
sudo apt install llvm-8-dev libclang-8-dev clang-8 xz-utils lld
```

#### Running Regression Tests

When running presubmits, the fuzz tests are run **without** a fuzzing engine.
Only the corpus data will be used for each fuzz test.
Therefore, the corpus should contain at least a few files that are known to cause bugs.
This serves as regression tests.

You can run these regression tests locally using blaze. For instance:

```.shell script
blaze test -c opt --test_output=all //src/envoy/utils:json_struct_fuzz_test
```

#### Mutation and Generation Tests

When running continuously, the fuzz tests are run with a fuzzing engine to discover new bugs.
We intend the tests to be run with [LibFuzzer](https://llvm.org/docs/LibFuzzer.html), but are compatible with other engines.

LibFuzzer is a coverage-guided, evolutionary fuzzing engine.
Therefore, the corpus should contain a few inputs that LibFuzzer can modify.
When a bug is discovered via continuous fuzzing, the input data should be added to the corresponding test's corpus to serve as a regression test in presubmits.

You can run the fuzz test with the fuzz engine locally using blaze.
Note the `_with_libfuzzer` suffix on the target under test.
For instance:

```.shell script
blaze test --config=asan-fuzzer \
           --test_arg="${ROOT}/tests/fuzz/corpus/json_struct" \
           --test_arg="-max_total_time=15" \
           --test_output=streamed \
           //src/envoy/utils:json_struct_fuzz_test_with_libfuzzer
```

To understand the output of the fuzzer, see [this documentation](https://llvm.org/docs/LibFuzzer.html#output).

The fuzzer will not write newly generated corpus entries to your working directory.
To run the fuzzer and generate new corpus entries, use `blaze run` instead:

```.shell script
blaze run --config=asan-fuzzer \
          --test_output=streamed \
          //src/envoy/utils:json_struct_fuzz_test_with_libfuzzer \
          ${ROOT}/tests/fuzz/corpus/json_struct \
          -max_total_time=15
```

Don't let it run for too long, as a lot of entries will be generated.
Please do not commit generated entries to git.