package()

# To generate localhost.crt and localhost.key
# openssl req -newkey rsa:2048 \
#  -new -nodes -x509 \
#  -days 3650 \
#  -out server.crt \
#  -keyout server.key \
#  -subj "/C=US/ST=California/L=Mountain View/O=Your Organization/OU=Your Unit/CN=localhost"

exports_files([
    "server.crt",
    "server.key",
    "proxy.crt",
    "proxy.key",
    "logs_metrics.pb.txt",
])
