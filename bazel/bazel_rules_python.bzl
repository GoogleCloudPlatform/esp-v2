# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

load(
    "@bazel_tools//tools/build_defs/repo:git.bzl",
    "git_repository",
)

def bazel_rules_python_repositories(load_repo = True):
    if load_repo:
        git_repository(
            name = "io_bazel_rules_python",
            commit = "8b5d0683a7d878b28fffe464779c8a53659fc645",
            remote = "https://github.com/bazelbuild/rules_python.git",
        )

def paths(files):
    return [f.path for f in files]

def _impl(ctx):
    descriptors = []
    for pl in ctx.attr.proto_libraries:
        descriptors += list(pl.proto.transitive_descriptor_sets)
    ctx.actions.run_shell(
        inputs = descriptors,
        outputs = [ctx.outputs.out],
        command = "cat %s > %s" % (
            " ".join(paths(descriptors)),
            ctx.outputs.out.path,
        ),
    )

gen_proto_descriptors = rule(
    implementation = _impl,
    attrs = {
        "proto_libraries": attr.label_list(
            allow_files = False,
            mandatory = True,
        ),
        "out": attr.output(mandatory = True),
    },
)