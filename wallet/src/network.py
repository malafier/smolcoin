import dataclasses
import json

import requests

from src.transactions import TransactionMessage


def send_message(address: str, message: str):
    requests.post(url=f"http://{address}/broadcast", json={"message": message})


def send_transaction(address: str, transaction: TransactionMessage):
    data = dataclasses.asdict(transaction)
    # print(data)
    requests.post(
        url=f"http://{address}/transaction",
        headers={"Content-Type": "application/json"},
        json=data,
    )


def get_possible_pubs(address: str) -> list[str]:
    resp = requests.get(
        url=f"http://{address}/ids",
    )
    ids_str = resp.json()
    return json.loads(ids_str)["ids"]
