#!/usr/bin/env python3

import json
import logging
import os
import re
import sys

import msal
import requests


MRSSANTA_API_URL = os.getenv("MRSSANTA_API_URL")
MRSSANTA_API_KEY = os.getenv("MRSSANTA_API_KEY")
ENTRA_GROUPS_ENV = "ENTRA_GROUPS"

GRAPH_BASE_URL = "https://graph.microsoft.com/v1.0"
GRAPH_SCOPE = ["https://graph.microsoft.com/.default"]
GRAPH_TIMEOUT_SECONDS = 60
INCLUDE_NESTED_GROUPS = True

logger = logging.getLogger(__name__)


class GraphError(Exception):
    pass


def configure_logging():
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(message)s",
    )


def get_required_env(name, fallback_name=None):
    value = os.getenv(name)
    if not value and fallback_name:
        value = os.getenv(fallback_name)
    if not value:
        names = f"{name} or {fallback_name}" if fallback_name else name
        raise SystemExit(f"Missing required environment variable: {names}")
    return value


def get_env_list(name):
    value = get_required_env(name).strip()
    if value[:1] == value[-1:] and value[:1] in {"'", '"'}:
        value = value[1:-1].strip()

    try:
        items = json.loads(value)
    except json.JSONDecodeError:
        items = value.split(",")

    if not isinstance(items, list):
        raise SystemExit(
            f"{name} must be a JSON array or a comma-separated list."
        )

    for item in items:
        if not isinstance(item, str):
            raise SystemExit(f"{name} must only contain strings.")

    return items


def get_access_token():
    client_id = get_required_env("AZURE_CLIENT_ID")
    client_secret = get_required_env("AZURE_CLIENT_SECRET")
    tenant_id = get_required_env("AZURE_TENANT_ID")

    authority = f"https://login.microsoftonline.com/{tenant_id}"
    client = msal.ConfidentialClientApplication(
        client_id,
        authority=authority,
        client_credential=client_secret,
    )

    token_result = client.acquire_token_silent(GRAPH_SCOPE, account=None)
    if not token_result:
        token_result = client.acquire_token_for_client(scopes=GRAPH_SCOPE)

    if "access_token" not in token_result:
        error = token_result.get("error", "unknown_error")
        description = token_result.get(
            "error_description",
            "No error description returned.",
        )
        raise SystemExit(f"Could not acquire Graph token: {error}: {description}")

    return token_result["access_token"]


def graph_get(session, access_token, url, params=None):
    headers = {
        "Authorization": f"Bearer {access_token}",
        "Accept": "application/json",
    }
    response = session.get(
        url,
        headers=headers,
        params=params,
        timeout=GRAPH_TIMEOUT_SECONDS,
    )
    if not response.ok:
        raise GraphError(f"{response.status_code} from {url}: {response.text}")
    return response.json()


def get_paged_values(session, access_token, url, params=None):
    values = []
    next_url = url
    next_params = params

    while next_url:
        data = graph_get(session, access_token, next_url, next_params)
        values.extend(data.get("value", []))
        next_url = data.get("@odata.nextLink")
        next_params = None

    return values


def is_object_id(value):
    return bool(
        re.fullmatch(
            r"[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-"
            r"[0-9a-fA-F]{4}-[0-9a-fA-F]{12}",
            value,
        )
    )


def escape_odata_string(value):
    return value.replace("'", "''")


def dedupe_by_id(items):
    deduped = {}
    for item in items:
        item_id = item.get("id")
        if item_id:
            deduped[item_id] = item
    return list(deduped.values())


def resolve_group(session, access_token, group_name_or_id):
    group_select = "id,displayName,mail,mailNickname"

    if is_object_id(group_name_or_id):
        return graph_get(
            session,
            access_token,
            f"{GRAPH_BASE_URL}/groups/{group_name_or_id}",
            {"$select": group_select},
        )

    escaped_value = escape_odata_string(group_name_or_id)
    matches = []
    for field in ["displayName", "mailNickname", "mail"]:
        matches.extend(
            get_paged_values(
                session,
                access_token,
                f"{GRAPH_BASE_URL}/groups",
                {
                    "$filter": f"{field} eq '{escaped_value}'",
                    "$select": group_select,
                    "$top": "999",
                },
            )
        )

    matches = dedupe_by_id(matches)
    if not matches:
        raise GraphError(f"No group found matching: {group_name_or_id}")
    if len(matches) > 1:
        names = ", ".join(
            f"{group.get('displayName')} ({group.get('id')})"
            for group in matches
        )
        raise GraphError(
            f"Multiple groups matched {group_name_or_id}. Use an object ID: {names}"
        )

    return matches[0]


def get_group_user_members(session, access_token, group_id):
    relation = "transitiveMembers" if INCLUDE_NESTED_GROUPS else "members"
    return get_paged_values(
        session,
        access_token,
        f"{GRAPH_BASE_URL}/groups/{group_id}/{relation}/microsoft.graph.user",
        {
            "$select": "id,displayName,mail,userPrincipalName,accountEnabled",
            "$top": "999",
        },
    )


def get_configured_groups():
    groups = []
    seen = set()

    for group in get_env_list(ENTRA_GROUPS_ENV):
        normalized = group.strip()
        if normalized and normalized not in seen:
            groups.append(normalized)
            seen.add(normalized)

    if not groups:
        raise SystemExit(f"No Entra groups configured in {ENTRA_GROUPS_ENV}.")

    return groups


def collect_users_from_groups(session, access_token):
    users_by_id = {}

    for group_input in get_configured_groups():
        group = resolve_group(session, access_token, group_input)
        group_name = group.get("displayName") or group_input
        logger.info("Fetching users from %s", group_name)

        group_users = get_group_user_members(
            session,
            access_token,
            group["id"],
        )
        logger.info("Found %d users in %s", len(group_users), group_name)

        for user in group_users:
            user_id = user.get("id")
            if not user_id:
                continue

            stored_user = users_by_id.setdefault(
                user_id,
                {
                    "id": user_id,
                    "displayName": user.get("displayName"),
                    "mail": user.get("mail"),
                    "userPrincipalName": user.get("userPrincipalName"),
                    "accountEnabled": user.get("accountEnabled"),
                    "groups": [],
                },
            )
            stored_user["groups"].append(group_name)

    users = list(users_by_id.values())
    return sorted(
        users,
        key=lambda user: (
            user.get("userPrincipalName")
            or user.get("mail")
            or user["id"]
        ),
    )

def get_mrssanta_devices(url, key):
    url = f"{url}/api/devices"
    logger.info(f"GETing events from {url}")
    headers = {
        "X-API-Key": f"{key}",
    }
    result = requests.get(url, headers=headers, timeout=60)
    if result.status_code != 200:
        logger.info(f"Error fetching events from {url}: {result.status_code} {result.text}")
        sys.exit(1)
    return result.json()


def post_mrssanta_device(url, key, data):
    url = f"{url}/api/devices"
    logger.info(f"POSTing to {url}")
    headers = {
        "X-API-Key": f"{key}",
        "Content-Type": "application/json"
    }
    post_data = json.dumps(data)
    result = requests.post(url, headers=headers, data=post_data)
    if result.status_code != 200:
        logger.info(f"Error posting devices to {url}: {result.status_code} {result.text}")
        sys.exit(1)
    else:
        logger.info(f"Sucessfully posting devices to {url}: {result.status_code} {result.text}")


def find_mrssanta_devices(users):
    results = []
    # Add the next part of the script here. `users` is a deduplicated list of
    # Entra users collected from every group in ENTRA_GROUPS.
    devices = get_mrssanta_devices(MRSSANTA_API_URL, MRSSANTA_API_KEY)
    logger.info(f"Fetched {len(devices)} devices from MrsSanta")
    for user in users:
        username = f"{user['userPrincipalName'].split('@')[0]}"
        matches = [
            item for item in devices
            if item.get("primary_user") == username
        ]

        if matches:
            results.extend(matches)

    return results


def main():
    configure_logging()

    try:
        access_token = get_access_token()
        with requests.Session() as session:
            users = collect_users_from_groups(session, access_token)
    except GraphError as error:
        raise SystemExit(f"Graph request failed: {error}") from error

    logger.info("Collected %d unique users", len(users))

    devices = find_mrssanta_devices(users)
    logger.info("Found %d matching devices in MrsSanta", len(devices))

    devices_to_update = [
        device for device in devices
        if device.get("client_mode") != 1 or device.get("primary_user_role") != "Developer"
    ]

    for device in devices_to_update:
        device["client_mode"] = 1
        device["primary_user_role"] = "Developer"


    logger.debug(json.dumps(devices_to_update, indent=2))

    logger.info("Changing %d matching devices in MrsSanta", len(devices_to_update))

    if len(devices_to_update) > 0:
        data = []
        for device in devices_to_update:
            keys = ["last_clean_sync", "last_updated", "last_sync"]
            removed = {}
            for key in keys:
                if key in device:
                    removed[key] = device.pop(key)
            data.append(device)
        post_mrssanta_device(MRSSANTA_API_URL, MRSSANTA_API_KEY, data)


if __name__ == "__main__":
    main()
