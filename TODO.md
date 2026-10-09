Use qpdf for better text extraction from PDFs.

See if interning strings improves efficiency.

Store the file offset of each token in the index, so we can print snippets of
text that match the query.

Add an option to match substrings?

Store file hashes and mtimes, and a real database, so that we can read and
update the database incrementally instead of all at once every time. And we
won’t have to maintain the entire database in memory while indexing.

Test multi-word queries more thoroughly.

Better tokenization, including enabling multiple outputs from a single token
(e.g. “dogs” should yield both “dogs” and “dog”, and maybe “santé” should yield
a normalized form of “santé” and “sante”).

Also multi-tokenize numbers.

Keep a tally of bytes indexed per second in `Index`.

Design and implement fancier querying syntax and semantics.

Improve test coverage. Make the tests exhaustively search for every word (and
every token derivable from it) in the testdata set.
