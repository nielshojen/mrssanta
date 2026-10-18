import os
import tempfile

# main.py configures Flask-Session at import time; keep its files out of the
# source tree while testing.
os.environ.setdefault("SESSION_FILE_DIR", tempfile.mkdtemp(prefix="mrssanta-admin-sessions-"))
os.environ.setdefault("FLASK_SECRET_KEY", "test-secret")
