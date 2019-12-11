package()

# To generate localhost.crt and localhost.key
# openssl req -newkey rsa:2048 \
#  -new -nodes -x509 \
#  -days 3650 \
#  -out localhost.crt \
#  -keyout localhost.key \
#  -subj "/C=US/ST=California/L=Mountain View/O=Your Organization/OU=Your Unit/CN=localhost"

exports_files([
    "localhost.crt",
    "localhost.key",
    "logs_metrics.pb.txt",
])