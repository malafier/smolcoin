import requests


def broadcast_message(address: str, message: str):
    requests.post(url=f"http://{address}/broadcast", json={"message": message})
