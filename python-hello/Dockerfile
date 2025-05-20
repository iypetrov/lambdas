FROM public.ecr.aws/lambda/python:3.13
WORKDIR /app
COPY . .
RUN pip install -r requirements.txt
ENTRYPOINT ["python3", "-m", "awslambdaric"]
CMD ["app.handler"]
