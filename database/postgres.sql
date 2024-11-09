CREATE TABLE "url" (
    "id"     BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    "slug"   TEXT NOT NULL,
    "target" TEXT NOT NULL,
    "hits"   INTEGER DEFAULT 0
);

CREATE UNIQUE INDEX "url_slug_uniq_idx" ON "url" ( "slug" );
