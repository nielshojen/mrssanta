import os

from flask import Flask, render_template, send_from_directory
from google.cloud import firestore

app = Flask(__name__)

DATABASE_NAME = os.getenv("FIRESTORE_DATABASE", "default")

db = firestore.Client(database=DATABASE_NAME)

def get_firestore_count(collection_name):
    count_query = db.collection(collection_name).count()
    count_result = list(count_query.get())
    return count_result[0][0].value if count_result else 0

@app.route('/site.webmanifest')
def manifest():
    return send_from_directory('static/manifest', 'site.webmanifest', mimetype='application/manifest+json')

@app.route("/")
def index():
    collections_to_count = {
        "devices": {"collection": f"{DATABASE_NAME}_devices"},
        "rules": {"collection": f"{DATABASE_NAME}_rules"},
        "events": {"collection": f"{DATABASE_NAME}_events"},
    }

    counts = {
        db_key: get_firestore_count(db_info["collection"])
        for db_key, db_info in collections_to_count.items()
    }

    return render_template("index.html", counts=counts)

@app.route("/devices")
def devices():
    docs = db.collection(f"{DATABASE_NAME}_devices").stream()
    data = [doc.to_dict() for doc in docs]

    return render_template("devices.html", data=data)

@app.route("/events")
def events():
    docs = db.collection(f"{DATABASE_NAME}_events").stream()
    data = [doc.to_dict() for doc in docs]

    return render_template("events.html", data=data)

@app.route("/rules")
def rules():
    docs = db.collection(f"{DATABASE_NAME}_rules").stream()
    data = [doc.to_dict() for doc in docs]

    return render_template("rules.html", data=data)

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)