# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM node:0.12
COPY tests/e2e/testdata/bookstore/http_bookstore.js /http_bookstore.js
COPY tests/e2e/testdata/bookstore/bookstore_swagger_template.json /bookstore_swagger_template.json
COPY tests/e2e/testdata/bookstore/package.json /package.json
RUN npm install
CMD PORT=8081 node http_bookstore.js