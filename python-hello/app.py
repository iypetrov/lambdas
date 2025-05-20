import json
from config import Config
from logger import ContextLogger

config = Config()
logger = ContextLogger(config)


def handler(event, context):
    logger.info("Hello, received event is: %s", json.dumps(event))
    return {
        "statusCode": 200,
        "body": json.dumps(event)
    }
