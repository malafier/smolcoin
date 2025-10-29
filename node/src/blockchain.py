import json

from cryptography.hazmat.primitives import hashes


class Block:
    def __init__(self, _id: int, previous_hash: str, timestamp: int, data: str) -> None:
        self.id: int = _id
        self.previous_hash: str = previous_hash
        self.timstamp: int = timestamp
        self.data: str = data

        self.hash: str = self.to_hash()

    def to_hash(self) -> str:
        data = {
            "id": self.id,
            "previous_hash": self.previous_hash,
            "timestamp": self.timstamp,
            "data": self.data,
        }
        block_bytes = json.dumps(data, sort_keys=True).encode("utf-8")

        hash_fun = hashes.Hash(hashes.SHA256())
        hash_fun.update(block_bytes)
        hash_bytes = hash_fun.finalize()
        return hash_bytes.hex()
