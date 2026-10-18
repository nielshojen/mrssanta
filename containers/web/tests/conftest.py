import os
import tempfile

# main.py reads its configuration and sets up Flask-Session at import time;
# give it test values and keep session files out of the source tree.
os.environ.setdefault("SESSION_FILE_DIR", tempfile.mkdtemp(prefix="mrssanta-web-sessions-"))
os.environ.setdefault("FLASK_SECRET_KEY", "test-secret")
os.environ.setdefault("MONGO_DB", "mrssanta-test")
os.environ.setdefault("VOTE_THRESHOLD", "3")
os.environ.setdefault("SYNC_INTERVAL_LOCKDOWN", "60")
