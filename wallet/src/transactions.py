import json
from dataclasses import dataclass

from src.crypto import sign


@dataclass
class Transaction:
    sender: str
    receiver: str
    amount: float
    timestamp: int
    signature: str

    def __init__(self) -> None:
        self.sender = ""
        self.receiver = ""
        self.amount = 0.0
        self.timestamp = 0

    def sign(self, sk: str):
        tx = {
            "sender": self.sender,
            "receiver": self.receiver,
            "amount": int(self.amount) if self.amount.is_integer() else self.amount,
            "timestamp": self.timestamp,
        }
        tx_str = json.dumps(tx, separators=(",", ":"))
        self.signature = sign(sk, tx_str).hex()
