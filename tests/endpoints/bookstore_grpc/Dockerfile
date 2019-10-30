# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# docker build -f tests/endpoints/bookstore_grpc/Dockerfile -t bookstore .
# docker run --rm -it -p 8082:8082 bookstore

FROM gcr.io/google_appengine/nodejs

ADD tests/endpoints/bookstore_grpc /app
WORKDIR /app

RUN npm install

ENTRYPOINT []

EXPOSE 8082
CMD ["npm", "start"]