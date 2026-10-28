
import json
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

from pathlib import Path
from shutil import copyfileobj
from urllib.parse import urlencode


class Client:
    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url.rstrip("/")

    
    def _request(self, method, path, data=None, binary=False):
        body = None

        if data is not None:
            if binary:
                body = data
            else:
                body = json.dumps(data, allow_nan=False).encode("utf-8")

        request = Request(
            self.base_url + path,
            data=body,
            headers={
                "Content-Type": (
                    "application/octet-stream" if binary else "application/json"
                )
            },
            method=method,
        )

        try:
            with urlopen(request, timeout=60 if binary else 10) as response:
                return json.load(response)

        except HTTPError as error:
            try:
                message = json.load(error).get("error", error.reason)
            except ValueError:
                message = error.reason
            raise RuntimeError(
                f"{method} {path} failed (HTTP {error.code}): {message}"
            ) from error

        except URLError as error:
            raise ConnectionError(
                f"Cannot reach {self.base_url}: {error.reason}"
            ) from error


    def create_experiment(self, name, description=""):
        return self._request("POST", "/experiments", {
            "name": name,
            "description": description,
        })

    def start_run(self, experiment_id, name):
        data = self._request("POST", "/runs", {
            "experiment_id": experiment_id,
            "name": name,
        })

        return Run(self, data["id"])

    def list_runs(self, experiment_id):
        return self._request(
            "GET", f"/runs?experiment_id={experiment_id}"
        )

    def list_metrics(self, run_id):
        return self._request(
            "GET", f"/metrics?run_id={run_id}"
        )


class Run:
    def __init__(self, client, run_id):
        self.client = client
        self.id = run_id

    def log_param(self, key, value):
        return self.client._request("POST", "/params", {
            "run_id": self.id,
            "key": key,
            "value": str(value),
        })

    def log_metric(self, name, value, step=0):
        return self.client._request("POST", "/metrics", {
            "run_id": self.id,
            "name": name,
            "value": value,
            "step": step,
        })

    def finish(self, status="completed"):
        return self.client._request("PATCH", "/runs", {
            "id": self.id,
            "status": status,
        })

    
    def log_artifact(self, path):
        path = Path(path)
        query = urlencode({"run_id": self.id, "name": path.name})

        with path.open("rb") as file:
            return self.client._request(
                "POST", f"/artifacts?{query}", file, binary=True
            )

    def list_artifacts(self):
        return self.client._request(
            "GET", f"/artifacts?run_id={self.id}"
        )

    def download_artifact(self, name, path):
        query = urlencode({"run_id": self.id, "name": name})

        with urlopen(
            f"{self.client.base_url}/artifacts?{query}", timeout=60
        ) as response:
            with open(path, "wb") as file:
                copyfileobj(response, file)

