import pytest
from jinja2 import Environment, FileSystemLoader, select_autoescape

LOGO_URL = "/santa/static/android-chrome-192x192.png"


@pytest.fixture
def render():
    env = Environment(loader=FileSystemLoader("templates"), autoescape=select_autoescape())

    def _render(name, response, user=None):
        session = {"user": user} if user else {}
        return env.get_template(name).render(response=response, session=session)

    return _render


@pytest.fixture
def base_response():
    return {
        "organization": "Acme",
        "selfserviceapp": "Managed Software Center",
        "support_email": "help@example.com",
        "support_phone": "+45 1234",
        "vote_threshold": 3,
        "sync_interval": 1,
        "called_url": "/santa/blockables/abc",
    }


def binary_response(base_response, virustotal_result):
    # The server decides whether the Allow button is offered (voting.allow_decision).
    if virustotal_result == 2:
        allow = {"allowed": False, "reason": "Tool cannot be allowed because it was reported as malicious."}
    else:
        allow = {"allowed": True, "reason": "", "action": "new", "machine_id": "dev-1"}

    return {
        **base_response,
        "binary": {
            "file_name": "Tool",
            "file_path": "/Applications/Tool.app",
            "file_sha256": "abc123",
            "virustotal_result": virustotal_result,
        },
        "device": {"identifier": "dev-1", "hostname": "mac-01.example.com"},
        "ruleexists": False,
        "rule": {"identifier": "abc123", "rule_type": "BINARY"},
        "allow": allow,
    }


@pytest.mark.parametrize("name", ["moreinfo.html", "error.html", "404.html", "blockables.html"])
def test_pages_use_shared_theme(render, base_response, name):
    html = render(name, base_response)

    assert '<header class="app-header">' in html
    assert f'src="{LOGO_URL}" class="brand-mark"' in html
    assert '<meta name="theme-color" content="#ff5050">' in html
    assert "--santa-logo-red: #ff5050;" in html
    assert 'data-bs-theme="dark"' in html


def test_moreinfo_uses_themed_inset_panel(render, base_response):
    html = render("moreinfo.html", base_response)

    assert 'class="inset-panel p-3"' in html
    assert "bg-light border rounded" not in html


def test_error_retry_uses_primary_button(render, base_response):
    html = render("error.html", base_response)

    assert '<a href="/santa/blockables/abc" class="btn btn-primary">Try Again</a>' in html


def test_user_shown_in_header(render, base_response):
    html = render("moreinfo.html", base_response, user={"name": "Ada"})

    assert "Ada" in html


def test_safe_binary_can_be_allowed(render, base_response):
    html = render("blockables.html", binary_response(base_response, 1))

    assert "data-allow-rule" in html
    assert 'text-bg-success">Safe' in html
    assert '"machine_id": "dev-1"' in html


def test_malicious_binary_cannot_be_allowed(render, base_response):
    html = render("blockables.html", binary_response(base_response, 2))

    assert "data-allow-rule" not in html
    assert "status-strip is-danger" in html
    assert "Cannot Add Rule" in html


def test_refused_request_shows_reason(render, base_response):
    response = binary_response(base_response, 1)
    response["allow"] = {"allowed": False, "reason": "You can only request access for your own Mac."}

    html = render("blockables.html", response)

    assert "data-allow-rule" not in html
    assert "You can only request access for your own Mac." in html
