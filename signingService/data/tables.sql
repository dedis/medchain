CREATE TABLE Action(
  id TEXT PRIMARY KEY,
  initiator TEXT,
  status TEXT DEFAULT "WAITING",
  action_value TEXT
);

CREATE TABLE SignatureStatus(
  action_id TEXT,
  signer_identity TEXT,
  signed INTEGER DEFAULT 0,
  FOREIGN KEY(action_id) REFERENCES Action(id),
  PRIMARY KEY(action_id, signer_identity)
);
