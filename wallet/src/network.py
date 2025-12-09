import dataclasses

import requests

from src.transactions import Transaction


def send_transaction(address: str, tx: Transaction):
    data = dataclasses.asdict(tx)
    print(data)
    requests.post(
        url=f"http://{address}/transaction",
        headers={"Content-Type": "application/json"},
        json=data,
    )


def get_possible_ids(address: str) -> list[str]:
    resp = requests.get(
        url=f"http://{address}/users",
    )
    return resp.json()


def get_ledger(address: str, id: str) -> dict[str, float]:
    resp = requests.get(
        url=f"http://{address}/ledger",
        headers={"Content-Type": "application/json"},
        json={"id": id},
    )
    if resp.status_code != 200:
        print(f"Err: {resp.status_code} {resp.text}")
    return resp.json()
