import os
import requests
import math
import msal
import base64
from urllib.parse import unquote, urlsplit
from datetime import datetime, timezone

from flask import Flask, jsonify, render_template, request, send_from_directory, redirect, session, url_for
from flask_session import Session
from markupsafe import escape
from pymongo import MongoClient

from log_config import configure_logging
import voting


logger = configure_logging("web", "mrssanta-web")

app = Flask(__name__, static_folder=None)

app.logger.handlers.clear()
app.logger.propagate = True

app.secret_key = os.getenv("FLASK_SECRET_KEY")

app.config["SESSION_TYPE"] = "filesystem"
app.config["SESSION_FILE_DIR"] = os.getenv("SESSION_FILE_DIR") or os.path.join(app.root_path, "flask_session")

Session(app)

CLIENT_ID = os.getenv("MSAL_CLIENT_ID")
CLIENT_SECRET = os.getenv("MSAL_CLIENT_SECRET")
TENANT_ID = os.getenv("MSAL_TENANT_ID")
AUTHORITY = f"https://login.microsoftonline.com/{TENANT_ID}"
REDIRECT_URI = os.getenv("MSAL_REDIRECT_URI")
SCOPES = ["User.Read"]
NEXT_URL_SESSION_KEY = "next_url"
IGNORED_NAVIGATION_PATHS = {
    "/santa/login",
    "/santa/token",
    "/santa/logout",
}

organization = os.environ.get('ORGANIZATION', 'this Organization')
selfserviceapp = os.environ.get('SELF_SERVICE_APP', 'Managed Software Center')
support_email = os.environ.get('SUPPORT_EMAIL', None)
support_phone = os.environ.get('SUPPORT_PHONE', None)
vt_api_key = os.environ.get('VT_API_KEY')
vote_threshold = int(os.environ.get('VOTE_THRESHOLD'))
sync_interval = int(os.environ.get('SYNC_INTERVAL_LOCKDOWN'))

# MongoDB connection
MONGO_URI = os.environ.get("MONGO_URI")
client = MongoClient(MONGO_URI)

# Select database
db = client[os.environ.get("MONGO_DB")]

STATIC_FOLDER = os.path.join(app.root_path, "static")


def get_current_url_path():
    if request.query_string:
        return f"{request.path}?{request.query_string.decode('utf-8')}"
    return request.path


def is_safe_redirect_url(target):
    if not target:
        return False

    parsed = urlsplit(target)
    if parsed.scheme or parsed.netloc:
        return False

    return parsed.path.startswith("/santa/") and parsed.path not in IGNORED_NAVIGATION_PATHS


def remember_next_url(target=None):
    target = target or get_current_url_path()
    if is_safe_redirect_url(target):
        session[NEXT_URL_SESSION_KEY] = target
        return target
    return None


def redirect_to_login():
    next_url = remember_next_url()
    if next_url:
        return redirect(url_for("login", next=next_url))
    return redirect(url_for("login"))


def get_post_login_redirect(target=None):
    session_target = session.pop(NEXT_URL_SESSION_KEY, None)
    for redirect_target in (target, session_target):
        if is_safe_redirect_url(redirect_target):
            return redirect_target
    return url_for("moreinfo")


def get_internal_referrer_url():
    referrer = request.referrer
    if not referrer:
        return None

    parsed = urlsplit(referrer)
    if parsed.scheme and not parsed.netloc:
        return None

    if parsed.netloc and parsed.netloc != request.host:
        return None

    if parsed.path == request.path:
        return None

    referrer_path = parsed.path
    if parsed.query:
        referrer_path = f"{referrer_path}?{parsed.query}"

    if not is_safe_redirect_url(referrer_path):
        return None

    return referrer_path


def create_template_response():
    return {
        "called_url": request.url,
    }


def get_blockables_path_params(request):
    path = unquote((request.path or "").strip("/"))
    parts = path.split("/")

    if len(parts) == 4 and parts[0] == "santa" and parts[1] == "blockables":
        return parts[2] or None, parts[3] or None

    return None, None

def get_vt_result(file_hash):
    check_failed = 0

    url = "https://www.virustotal.com/api/v3/files/%s" % file_hash

    headers = {"accept": "application/json", "x-apikey": vt_api_key}

    response = requests.get(url, headers=headers)

    result = response.json()

    if response.status_code == 200 and len(result['data']) > 0:
        try:
            malicious = result['data']['attributes']['last_analysis_stats']['malicious']
        except:
            malicious = 0
            check_failed = 1
    else:
        malicious = 0
        check_failed = 1

    if check_failed == 0:
        if malicious == 0:
            return 1
        else:
            return 2
    else:
        return 0


def get_device(identifier):
    """Fetch a device by its identifier from MongoDB."""
    collection = db["devices"]

    device = collection.find_one({"_id": identifier})

    return device if device else None


def get_binary(identifier):
    """Fetch a binary by its identifier from MongoDB and update VirusTotalResult if missing."""
    collection = db["events"]

    binary = collection.find_one({"_id": identifier})

    if binary:
        if "virustotal_result" not in binary or binary["virustotal_result"] == 0:
            virustotalresult = get_vt_result(binary.get("file_sha256"))
            if virustotalresult is not None:
                binary["virustotal_result"] = virustotalresult
                save_binary(identifier, binary)
        return binary
    else:
        return None


def save_binary(identifier, data):
    """Saves or updates a binary document in MongoDB."""
    collection = db["events"]

    data["last_updated"] = datetime.now(timezone.utc)

    collection.update_one(
        {"_id": identifier},
        {"$set": data},
        upsert=True
    )

    return data


def get_rule(identifier):
    """Fetch a rule by its identifier from MongoDB."""
    collection = db["rules"]

    rule = collection.find_one({"_id": identifier})

    return rule if rule else None


def save_rule(identifier, data):
    """Saves or updates a rule document in MongoDB."""
    collection = db["rules"]

    data["last_updated"] = datetime.now(timezone.utc)

    update_ops = {"$set": data}

    if "assigned" not in data:
        update_ops["$unset"] = {"assigned": ""}

    collection.update_one(
        {"_id": identifier},
        update_ops,
        upsert=True
    )

    return data


def save_cache(user_id, cache):
    if cache.has_state_changed:
        db["sessions"].update_one(
            {"user_id": user_id},
            {"$set": {
                "token_cache": cache.serialize(),
                "updated_at": datetime.utcnow()
            }},
            upsert=True
        )


@app.route("/health")
def status():
    return jsonify({"status": "ok"}), 200


# Icons and help images. Served by the app itself so it doesn't depend on
# a cloud storage bucket; the ingress routes /santa/static here.
@app.route("/santa/static/<path:filename>")
def static_files(filename):
    return send_from_directory(STATIC_FOLDER, filename, max_age=86400)


@app.route("/santa/login")
def login():
    next_url = remember_next_url(request.args.get("next"))

    # No user_id yet, so just create MSAL app without a token cache
    msal_app = msal.ConfidentialClientApplication(
        client_id=CLIENT_ID,
        authority=AUTHORITY,
        client_credential=CLIENT_SECRET,
    )
    auth_url_kwargs = {"redirect_uri": REDIRECT_URI}
    if next_url:
        auth_url_kwargs["state"] = next_url

    auth_url = msal_app.get_authorization_request_url(SCOPES, **auth_url_kwargs)
    return redirect(auth_url)


@app.route("/santa/token")
def token():
    """Handles login callback, gets user profile and profile picture."""

    # Error check
    if "error" in request.args:
        # Both values come from the query string, so anyone can put markup in
        # them with a crafted link: escape before returning HTML.
        return f"Error: {escape(request.args.get('error', ''))} - {escape(request.args.get('error_description', ''))}"

    # Code check
    code = request.args.get("code")
    if not code:
        return "No authorization code received"

    # Create fresh token cache
    cache = msal.SerializableTokenCache()

    # Build MSAL app with cache
    msal_app = msal.ConfidentialClientApplication(
        client_id=CLIENT_ID,
        authority=AUTHORITY,
        client_credential=CLIENT_SECRET,
        token_cache=cache
    )

    # Acquire token using auth code
    result = msal_app.acquire_token_by_authorization_code(
        code,
        scopes=SCOPES,
        redirect_uri=REDIRECT_URI
    )

    if "access_token" in result:
        access_token = result["access_token"]
        headers = {"Authorization": f"Bearer {access_token}"}

        # Fetch user profile
        user_info = requests.get("https://graph.microsoft.com/v1.0/me", headers=headers).json()
        user_id = user_info.get("id")

        # Fetch profile picture (optional)
        profile_pic = None
        pic_response = requests.get("https://graph.microsoft.com/v1.0/me/photo/$value", headers=headers)
        if pic_response.status_code == 200:
            profile_pic = base64.b64encode(pic_response.content).decode("utf-8")

        # Save MSAL token cache to MongoDB
        save_cache(user_id, cache)

        # Store user in session (for easy reference)
        session["user"] = {
            "id": user_id,
            "name": user_info.get("displayName", "User"),
            "email": user_info.get("mail", user_info.get("userPrincipalName")),
            "upn": user_info.get("userPrincipalName"),
            "profile_pic": profile_pic,
        }

        return redirect(get_post_login_redirect(request.args.get("state")))

    # If something went wrong
    return f"Authentication failed: {escape(result.get('error_description') or result.get('error') or '')}"


@app.route("/santa/logout")
def logout():
    session.clear()
    return redirect(url_for("login"))


@app.route("/santa/moreinfo")
def moreinfo():
    if "user" not in session:
        return redirect_to_login()

    response = create_template_response()
    response['organization'] = organization
    response['support_email'] = support_email
    response['support_phone'] = support_phone
    response['vote_threshold'] = vote_threshold
    response['sync_interval'] = math.ceil(sync_interval / 60)
    response['selfserviceapp'] = selfserviceapp
    response['back_url'] = get_internal_referrer_url()
    return render_template('moreinfo.html', response=response)


@app.route("/santa/blockables/<path:path>", methods=["GET", "POST"])
def blockables(path):
    if "user" not in session:
        return redirect_to_login()

    request_args = request.args
    path_machine_id, path_file_identifier = get_blockables_path_params(request)

    if request.method == "POST":
        # The request only names the device and the binary. Scope, rule type
        # and identifier are all derived on the server (see voting.py).
        data = request.get_json(silent=True) or {}
        machine_id = data.get("machine_id")
        file_sha256 = data.get("file_sha256")
        if not isinstance(machine_id, str) or not isinstance(file_sha256, str) or not machine_id or not file_sha256:
            return jsonify({"success": False, "message": "machine_id and file_sha256 are required"}), 400

        user = session["user"]
        device = get_device(machine_id)
        binary = get_binary(file_sha256)
        rule = voting.find_matching_rule(db, binary) if binary else None

        decision = voting.allow_decision(device, binary, rule, user)
        if not decision["allowed"]:
            logger.info("Refused blockables request", extra={"fields": {
                "machine_id": machine_id,
                "file_sha256": file_sha256,
                "user_id": user.get("id"),
                "reason": decision["reason"],
            }})
            return jsonify({"success": False, "message": decision["reason"]}), 403

        outcomes = voting.apply_allow_request(
            db, decision, binary, rule, user, vote_threshold, datetime.now(timezone.utc),
        )
        logger.info("Processed blockables request", extra={"fields": {
            "machine_id": machine_id,
            "file_sha256": file_sha256,
            "user_id": user.get("id"),
            "outcomes": outcomes,
        }})
        return jsonify({"success": True, "message": "Rule added successfully! Rule will be applied within a minute"}), 200

    if request.method == "GET":

        path = request.path.strip("/")

        path = unquote(path)

        if path.startswith("static/"):
            filename = path[len("static/"):]
            return send_from_directory(STATIC_FOLDER, filename)

        if path.startswith("favico.ico"):
            filename = "favico.ico"
            return send_from_directory(STATIC_FOLDER, filename)
    
        response = create_template_response()
        
        response['organization'] = organization
        response['selfserviceapp'] = selfserviceapp
        response['support_email'] = support_email
        response['support_phone'] = support_phone

        machine_id = None
        if request_args and "machine_id" in request_args:
            machine_id = request_args.getlist('machine_id')[0]
        elif path_machine_id:
            machine_id = path_machine_id

        if machine_id:
            device = get_device(machine_id)
            if device:
                response['device'] = device
            else:
                return render_template('error.html', response=response)

        file_identifier = None
        if request_args and "file_identifier" in request_args:
            file_identifier = request_args.getlist('file_identifier')[0]
        elif path_file_identifier:
            file_identifier = path_file_identifier

        if file_identifier:
            binary = get_binary(file_identifier)
            if binary:
                response['binary'] = binary
                rule = voting.find_matching_rule(db, binary)
                response['allow'] = voting.allow_decision(response.get('device'), binary, rule, session["user"])

                if rule:
                    response['ruleexists'] = True
                    response['rule'] = rule
                    if rule.get("scope") == "managedapp":
                        if machine_id and "assigned" in rule:
                            response["machine_assigned"] = machine_id in rule["assigned"]
                        else:
                            response["machine_assigned"] = False
                else:
                    response['ruleexists'] = False

                    target = voting.rule_target(binary)
                    rule = {"rule_type": target[0], "identifier": target[1]} if target else {}

                    response['rule'] = rule
            else:
                return render_template('error.html', response=response)
            

        return render_template('blockables.html', response=response)


@app.errorhandler(404)
def page_not_found(e):
    response = create_template_response()
    response['organization'] = organization
    response['support_email'] = support_email
    response['support_phone'] = support_phone
    return render_template('404.html', response=response), 404
                  

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 8080)))
