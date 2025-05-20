import logging


class ContextLogger:
    def __init__(self, config):
        self.logger = logging.getLogger("lambda_logger")
        handler = logging.StreamHandler()
        formatter = logging.Formatter("[%(levelname)s] %(message)s")
        handler.setFormatter(formatter)
        self.logger.addHandler(handler)
        self.logger.setLevel(logging.INFO)

        self.config = config

    def info(self, msg, *args):
        self.logger.info(f"[{self.config.app_env}] {msg}", *args)

    def error(self, msg, *args):
        self.logger.error(f"[{self.config.app_env}] {msg}", *args)
