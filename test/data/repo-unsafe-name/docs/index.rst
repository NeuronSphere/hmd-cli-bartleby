Test Unsafe Name
================

This repo has underscores in its manifest ``name`` field,
which would break LaTeX if used as-is in the document title.
The CLI should sanitize it automatically.
