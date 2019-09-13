# docker build -f tests/endpoints/bookstore_grpc/Dockerfile -t bookstore .
# docker run --rm -it -p 8082:8082 bookstore

FROM gcr.io/google_appengine/nodejs

ADD tests/endpoints/bookstore_grpc /app
WORKDIR /app

RUN npm install

ENTRYPOINT []

EXPOSE 8082
CMD ["npm", "start"]