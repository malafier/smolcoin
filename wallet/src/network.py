import dataclasses
import json

import requests

from src.transactions import Transaction


def send_transaction(address: str, tx: Transaction):
    data = dataclasses.asdict(tx)
    requests.post(
        url=f"http://{address}/transaction",
        headers={"Content-Type": "application/json"},
        json=data,
    )


def get_possible_ids(address: str) -> list[str]:
    resp = requests.get(
        url=f"http://{address}/ids",
    )
    return resp.json()


def get_ledger(address: str) -> dict[str, float]:
    resp = requests.get(
        url=f"http://{address}/ledger",
    )
    return resp.json()
