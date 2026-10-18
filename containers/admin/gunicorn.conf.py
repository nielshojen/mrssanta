import os

from log_config import make_gunicorn_logconfig


bind = f"0.0.0.0:{os.getenv('PORT', '8080')}"
workers = int(os.getenv("GUNICORN_WORKERS", "2"))
threads = int(os.getenv("GUNICORN_THREADS", "4"))
loglevel = os.getenv("GUNICORN_LOG_LEVEL", os.getenv("LOG_LEVEL", "INFO")).lower()

accesslog = "-"
errorlog = "-"
access_log_format = '%(h)s %(l)s %(u)s %(t)s "%(r)s" %(s)s %(b)s "%(f)s" "%(a)s"'
logconfig_dict = make_gunicorn_logconfig("admin", "mrssanta-admin")
