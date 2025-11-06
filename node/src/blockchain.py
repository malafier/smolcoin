import json
from dataclasses import dataclass

from cryptography.hazmat.primitives import hashes


def hash_sha256(data: bytes) -> str:
    hash_fun = hashes.Hash(hashes.SHA256())
    hash_fun.update(data)
    hash_bytes = hash_fun.finalize()
    return hash_bytes.hex()


class Block:
    def __init__(self, _id: int, previous_hash: str, timestamp: int, data: str) -> None:
        self.id: int = _id
        self.previous_hash: str = previous_hash
        self.timstamp: int = timestamp
        self.data: str = data

        self.nonce: int | None = None
        self.hash: str | None = None

    def mine(self, difficulty: int):
        nonce = 1
        prefix = "0" * difficulty

        while True:
            data = {
                "id": self.id,
                "previous_hash": self.previous_hash,
                "timestamp": self.timstamp,
                "data": self.data,
                "nonce": nonce,
            }
            block_bytes = json.dumps(data, sort_keys=True).encode("utf-8")
            block_hash = hash_sha256(block_bytes)

            if block_hash.startswith(prefix):
                self.nonce = nonce
                self.hash = block_hash
                return
            nonce += 1


@dataclass
class Transaction:
    def __init__(self, sender, reciever, ammount, difficulty, timestamp) -> None:
        self.sender: str = sender
        self.reciever: str = reciever
        self.ammount: float = ammount
        self.difficulty: int = difficulty
        self.timstamp: int = timestamp
