from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)


def test_regions_list():
    r = client.get("/v1/locations/regions")
    assert r.status_code == 200
    codes = {x["code"] for x in r.json()["items"]}
    assert {"AA", "OR", "AM", "TI", "SI", "DD"} <= codes
    assert "cities" not in r.json()["items"][0]


def test_region_with_cities():
    r = client.get("/v1/locations/regions/or")  # case-insensitive
    assert r.status_code == 200
    assert "Adama" in [c["name"]["en"] for c in r.json()["cities"]]


def test_city_by_name_and_slug():
    for key in ("Addis Ababa", "addis-ababa", "ADDIS ABABA"):
        r = client.get(f"/v1/locations/regions/AA/cities/{key}")
        assert r.status_code == 200, key
        subs = [s["name"]["en"] for s in r.json()["subcities"]]
        assert "Bole" in subs and len(subs) == 11


def test_unknown_city_and_region():
    assert client.get("/v1/locations/regions/AA/cities/Nairobi").status_code == 404
    r = client.get("/v1/locations/regions/XX")
    assert r.status_code == 404
    assert r.json()["error"]["code"] == "not_found"


def test_search_english_and_amharic():
    assert any(h["subcity"] == "Bole" for h in client.get("/v1/locations/search?q=bol").json()["items"] if h["type"] == "subcity")
    assert client.get("/v1/locations/search?q=ባሕር").json()["items"][0]["city"] == "Bahir Dar"


def test_demo_listing_locations_exist():
    for region, city in [("AA", "Addis Ababa"), ("OR", "Adama"), ("AM", "Bahir Dar"), ("OR", "Bishoftu"), ("SI", "Hawassa"),
                         ("DD", "Dire Dawa"), ("AM", "Debre Markos")]:
        assert client.get(f"/v1/locations/regions/{region}/cities/{city}").status_code == 200, (region, city)


def test_health_and_metrics():
    assert client.get("/healthz").json() == {"status": "ok"}
    assert client.get("/readyz").status_code == 200
    client.get("/v1/locations/regions")
    assert 'route="GET /v1/locations/regions"' in client.get("/metrics").text
