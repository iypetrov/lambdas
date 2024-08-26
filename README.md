# Test locally

- Setup AWS Lambda RIE https://github.com/aws/aws-lambda-runtime-interface-emulator 
- Build the docker image
```bash
docker build -t lambda-local:latest  .
```
- Run the docker image
```bash
docker run -p 9000:8080 --entrypoint /usr/local/bin/aws-lambda-rie lambda-local:latest ./bin/main
```
- Invoke the lambda
```bash
curl -XPOST "http://localhost:9000/2015-03-31/functions/function/invocations" -d '{"payload":"hello world!"}'
```