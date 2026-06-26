-- Colorblind-friendly color scheme preference (accessibility): 0 = default, 1 = on.
ALTER TABLE user ADD COLUMN colorblind INTEGER NOT NULL DEFAULT 0;
