import os


class Environment(str):
    LOCAL = "local"
    PROD = "prod"

    @classmethod
    def is_valid(cls, value):
        return value in (cls.LOCAL, cls.PROD)


class Config:
    def __init__(self):
        env = os.getenv("APP_ENV", Environment.PROD)
        if not Environment.is_valid(env):
            env = Environment.PROD
        self.app_env = env

    def __repr__(self):
        return f"<Config app_env={self.app_env}>"
