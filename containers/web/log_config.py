import json
import logging
import os
import sys
from datetime import datetime, timezone


class MaxLevelFilter(logging.Filter):
    def __init__(self, max_level):
        super().__init__()
        self.max_level = max_level

    def filter(self, record):
        return record.levelno <= self.max_level


class StructuredLogFormatter(logging.Formatter):
    def __init__(self, service, component):
        super().__init__()
        self.service = service
        self.component = component

    def format(self, record):
        severity = record.levelname
        entry = {
            "time": datetime.fromtimestamp(record.created, timezone.utc).isoformat().replace("+00:00", "Z"),
            "severity": severity,
            "level": severity.lower(),
            "status": "warn" if severity == "WARNING" else severity.lower(),
            "message": record.getMessage(),
            "service": self.service,
            "component": self.component,
        }

        env = os.getenv("ENVIRONMENT") or os.getenv("ENV") or os.getenv("DD_ENV")
        version = os.getenv("VERSION") or os.getenv("K_REVISION")
        if env:
            entry["env"] = env
        if version:
            entry["version"] = version

        fields = getattr(record, "fields", None)
        if isinstance(fields, dict) and fields:
            entry["fields"] = fields

        if record.exc_info:
            entry["exception"] = self.formatException(record.exc_info)

        return json.dumps(entry, default=str, separators=(",", ":"))


def get_service_name(default_service):
    return (
        os.getenv("SERVICE_NAME")
        or os.getenv("K_SERVICE")
        or os.getenv("DD_SERVICE")
        or default_service
    )


def get_log_level(*env_vars):
    level_name = "INFO"
    for env_var in env_vars or ("LOG_LEVEL",):
        value = os.getenv(env_var)
        if value:
            level_name = value.upper()
            break
    return getattr(logging, level_name, logging.INFO)


def configure_logging(component, default_service):
    service = get_service_name(default_service)
    level = get_log_level("LOG_LEVEL")
    formatter = StructuredLogFormatter(service, component)

    stdout_handler = logging.StreamHandler(sys.stdout)
    stdout_handler.setLevel(logging.DEBUG)
    stdout_handler.addFilter(MaxLevelFilter(logging.WARNING))
    stdout_handler.setFormatter(formatter)

    stderr_handler = logging.StreamHandler(sys.stderr)
    stderr_handler.setLevel(logging.ERROR)
    stderr_handler.setFormatter(formatter)

    root_logger = logging.getLogger()
    root_logger.handlers.clear()
    root_logger.setLevel(level)
    root_logger.addHandler(stdout_handler)
    root_logger.addHandler(stderr_handler)

    logger = logging.getLogger(component)
    logger.handlers.clear()
    logger.setLevel(level)
    logger.propagate = True
    return logger


def make_gunicorn_logconfig(component, default_service):
    service = get_service_name(default_service)
    level = get_log_level("GUNICORN_LOG_LEVEL", "LOG_LEVEL")

    return {
        "version": 1,
        "disable_existing_loggers": False,
        "filters": {
            "max_warning": {
                "()": MaxLevelFilter,
                "max_level": logging.WARNING,
            },
        },
        "formatters": {
            "structured": {
                "()": StructuredLogFormatter,
                "service": service,
                "component": component,
            },
            "access": {
                "()": StructuredLogFormatter,
                "service": service,
                "component": f"{component}.access",
            },
        },
        "handlers": {
            "stdout": {
                "class": "logging.StreamHandler",
                "level": "DEBUG",
                "filters": ["max_warning"],
                "formatter": "structured",
                "stream": "ext://sys.stdout",
            },
            "stderr": {
                "class": "logging.StreamHandler",
                "level": "ERROR",
                "formatter": "structured",
                "stream": "ext://sys.stderr",
            },
            "access_stdout": {
                "class": "logging.StreamHandler",
                "level": "INFO",
                "formatter": "access",
                "stream": "ext://sys.stdout",
            },
        },
        "root": {
            "level": level,
            "handlers": ["stdout", "stderr"],
        },
        "loggers": {
            "gunicorn.error": {
                "level": level,
                "handlers": ["stdout", "stderr"],
                "propagate": False,
            },
            "gunicorn.access": {
                "level": "INFO",
                "handlers": ["access_stdout"],
                "propagate": False,
            },
        },
    }
