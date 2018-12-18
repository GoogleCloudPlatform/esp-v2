# docker build -f tests/endpoints/bookstore-grpc/Dockerfile -t bookstore .
# docker run --rm -it -p 8082:8082 bookstore

FROM gcr.io/google_appengine/nodejs

ADD tests/endpoints/bookstore-grpc /app
WORKDIR /app

RUN npm install

ENTRYPOINT []

EXPOSE 8082
CMD ["npm", "start"]