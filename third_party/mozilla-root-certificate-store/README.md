The roots.pem file is periodically generated from
https://hg.mozilla.org/mozilla-central/raw-file/tip/security/nss/lib/ckfw/builtins/certdata.txt
using
https://github.com/agl/extract-nss-root-certs.

The gRPC team handles the generation of the file.
We copy the file directly from
https://github.com/grpc/grpc/tree/master/etc.