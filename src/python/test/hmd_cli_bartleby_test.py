from unittest.mock import patch, MagicMock, call
import pytest
import textwrap
from pathlib import Path
from hmd_cli_bartleby.controller import (
    LocalController,
    DEFAULT_BUILDERS,
    SOURCES_MARKER,
    SOURCES_STAGING_DIR,
    INDEXES_MARKERS,
    _get_sources,
    _generate_toctree_entries,
    _stage_sources,
    _inject_sources,
    _restore_index,
    _cleanup_staged_sources,
    _validate_source_paths,
)


class TestGetDocuments:
    def _make_controller(self):
        ctrl = object.__new__(LocalController)
        return ctrl

    @patch("hmd_cli_bartleby.controller.read_manifest", return_value={})
    def test_fallback_uses_default_builders(self, mock_manifest):
        ctrl = self._make_controller()
        result = ctrl._get_documents(root_doc="all", shell="all")
        assert result == {"index": {"builders": ["html", "pdf"], "root_doc": "index"}}

    @patch("hmd_cli_bartleby.controller.read_manifest", return_value={})
    def test_fallback_does_not_use_shell_arg(self, mock_manifest):
        ctrl = self._make_controller()
        result = ctrl._get_documents(root_doc="all", shell="html")
        assert result["index"]["builders"] == DEFAULT_BUILDERS

    @patch(
        "hmd_cli_bartleby.controller.read_manifest",
        return_value={
            "bartleby": {
                "roots": {
                    "guide": {
                        "builders": ["html"],
                        "root_doc": "guide_index",
                    },
                    "api": {
                        "builders": ["html", "pdf"],
                        "root_doc": "api_index",
                    },
                }
            }
        },
    )
    def test_manifest_roots_returned_as_is(self, mock_manifest):
        ctrl = self._make_controller()
        result = ctrl._get_documents(root_doc="all", shell="all")
        assert "guide" in result
        assert "api" in result
        assert result["guide"]["builders"] == ["html"]
        assert result["api"]["builders"] == ["html", "pdf"]

    @patch(
        "hmd_cli_bartleby.controller.read_manifest",
        return_value={
            "bartleby": {
                "roots": {
                    "guide": {
                        "builders": ["html"],
                        "root_doc": "guide_index",
                    },
                    "api": {
                        "builders": ["html", "pdf"],
                        "root_doc": "api_index",
                    },
                }
            }
        },
    )
    def test_specific_root_doc_filters(self, mock_manifest):
        ctrl = self._make_controller()
        result = ctrl._get_documents(root_doc="guide", shell="all")
        assert list(result.keys()) == ["guide"]

    @patch(
        "hmd_cli_bartleby.controller.read_manifest",
        return_value={
            "bartleby": {
                "roots": {
                    "guide": {"builders": ["html"], "root_doc": "guide_index"},
                    "api": {"builders": ["html", "pdf"], "root_doc": "api_index"},
                    "tutorial": {"builders": ["html"], "root_doc": "tutorial_index"},
                }
            }
        },
    )
    def test_comma_separated_multiple_roots(self, mock_manifest):
        ctrl = self._make_controller()
        result = ctrl._get_documents(root_doc="guide,api", shell="all")
        assert set(result.keys()) == {"guide", "api"}
        assert "tutorial" not in result

    @patch(
        "hmd_cli_bartleby.controller.read_manifest",
        return_value={
            "bartleby": {
                "roots": {
                    "guide": {"builders": ["html"], "root_doc": "guide_index"},
                    "api": {"builders": ["html", "pdf"], "root_doc": "api_index"},
                    "tutorial": {"builders": ["html"], "root_doc": "tutorial_index"},
                }
            }
        },
    )
    def test_comma_separated_with_whitespace(self, mock_manifest):
        ctrl = self._make_controller()
        result = ctrl._get_documents(root_doc="guide, api", shell="all")
        assert set(result.keys()) == {"guide", "api"}

    @patch(
        "hmd_cli_bartleby.controller.read_manifest",
        return_value={
            "bartleby": {
                "roots": {
                    "guide": {"builders": ["html"], "root_doc": "guide_index"},
                    "api": {"builders": ["html", "pdf"], "root_doc": "api_index"},
                    "tutorial": {"builders": ["html"], "root_doc": "tutorial_index"},
                }
            }
        },
    )
    def test_unknown_root_warns_but_continues(self, mock_manifest, capsys):
        ctrl = self._make_controller()
        result = ctrl._get_documents(root_doc="guide,nonexistent", shell="all")
        assert list(result.keys()) == ["guide"]
        captured = capsys.readouterr()
        assert "Warning: root document 'nonexistent' not found" in captured.out

    @patch(
        "hmd_cli_bartleby.controller.read_manifest",
        return_value={
            "bartleby": {
                "roots": {
                    "guide": {"builders": ["html"], "root_doc": "guide_index"},
                    "api": {"builders": ["html", "pdf"], "root_doc": "api_index"},
                    "tutorial": {"builders": ["html"], "root_doc": "tutorial_index"},
                }
            }
        },
    )
    def test_all_unknown_roots_raises(self, mock_manifest):
        ctrl = self._make_controller()
        with pytest.raises(SystemExit):
            ctrl._get_documents(root_doc="foo,bar", shell="all")

    @patch(
        "hmd_cli_bartleby.controller.read_manifest",
        return_value={
            "bartleby": {
                "roots": {
                    "guide": {"builders": ["html"], "root_doc": "guide_index"},
                    "api": {"builders": ["html", "pdf"], "root_doc": "api_index"},
                    "tutorial": {"builders": ["html"], "root_doc": "tutorial_index"},
                }
            }
        },
    )
    def test_trailing_comma_ignored(self, mock_manifest):
        ctrl = self._make_controller()
        result = ctrl._get_documents(root_doc="guide,", shell="all")
        assert list(result.keys()) == ["guide"]


class TestGetShells:
    def _make_controller(self):
        ctrl = object.__new__(LocalController)
        return ctrl

    @patch("hmd_cli_bartleby.controller.read_manifest", return_value={})
    def test_fallback_docs_produce_html_and_pdf_builds(self, mock_manifest):
        ctrl = self._make_controller()
        docs = {"index": {"builders": ["html", "pdf"], "root_doc": "index"}}
        builds = ctrl._get_shells(docs, shell="all")
        shells = [b["shell"] for b in builds]
        assert shells == ["html", "pdf"]

    @patch("hmd_cli_bartleby.controller.read_manifest", return_value={})
    def test_shell_flag_filters_builds(self, mock_manifest):
        ctrl = self._make_controller()
        docs = {"index": {"builders": ["html", "pdf"], "root_doc": "index"}}
        builds = ctrl._get_shells(docs, shell="html")
        assert len(builds) == 1
        assert builds[0]["shell"] == "html"

    @patch("hmd_cli_bartleby.controller.read_manifest", return_value={})
    def test_build_context_has_correct_fields(self, mock_manifest):
        ctrl = self._make_controller()
        docs = {"index": {"builders": ["html"], "root_doc": "index"}}
        builds = ctrl._get_shells(docs, shell="all")
        assert len(builds) == 1
        build = builds[0]
        assert build["name"] == "index"
        assert build["shell"] == "html"
        assert build["root_doc"] == "index"
        assert isinstance(build["config"], dict)

    @patch("hmd_cli_bartleby.controller.read_manifest", return_value={})
    def test_revealjs_shell_filters_builds(self, mock_manifest):
        ctrl = self._make_controller()
        docs = {"index": {"builders": ["html", "pdf", "revealjs"], "root_doc": "index"}}
        builds = ctrl._get_shells(docs, shell="revealjs")
        assert len(builds) == 1
        assert builds[0]["shell"] == "revealjs"

    @patch("hmd_cli_bartleby.controller.read_manifest", return_value={})
    def test_revealjs_in_multi_builder_doc(self, mock_manifest):
        ctrl = self._make_controller()
        docs = {
            "slides": {
                "builders": ["html", "revealjs"],
                "root_doc": "slides_index",
            },
            "guide": {
                "builders": ["html", "pdf"],
                "root_doc": "guide_index",
            },
        }
        builds = ctrl._get_shells(docs, shell="revealjs")
        assert len(builds) == 1
        assert builds[0]["name"] == "slides"
        assert builds[0]["shell"] == "revealjs"
        assert builds[0]["root_doc"] == "slides_index"


class TestGetSources:
    def test_empty_manifest(self):
        assert _get_sources({}) == {}

    def test_missing_sources_key(self):
        manifest = {"bartleby": {"roots": {"index": {}}}}
        assert _get_sources(manifest) == {}

    def test_valid_sources(self):
        manifest = {
            "bartleby": {
                "sources": {
                    "transform": {
                        "artifact_path": "target/artifacts/transform",
                        "docs_root": "docs",
                        "title": "Transform Service API",
                    }
                }
            }
        }
        result = _get_sources(manifest)
        assert "transform" in result
        assert result["transform"]["title"] == "Transform Service API"

    def test_missing_bartleby_key(self):
        manifest = {"build": {"commands": []}}
        assert _get_sources(manifest) == {}


class TestGenerateToctreeEntries:
    def test_empty_sources(self):
        assert _generate_toctree_entries({}) == ""

    def test_single_source_with_artifact_path(self):
        sources = {
            "transform": {
                "artifact_path": "target/artifacts/transform",
                "docs_root": "docs",
                "title": "Transform Service API",
            }
        }
        result = _generate_toctree_entries(sources)
        assert ".. toctree::" in result
        assert ":caption: Transform Service API" in result
        assert f"{SOURCES_STAGING_DIR}/transform/index" in result

    def test_multiple_sources_separate_toctrees(self):
        sources = {
            "transform": {
                "artifact_path": "target/artifacts/transform",
                "title": "Transform Service API",
            },
            "deployment": {
                "artifact_path": "target/artifacts/deployment",
                "title": "Deployment Service API",
            },
        }
        result = _generate_toctree_entries(sources)
        assert result.count(".. toctree::") == 2
        assert ":caption: Transform Service API" in result
        assert ":caption: Deployment Service API" in result

    def test_source_without_artifact_path(self):
        sources = {
            "existing": {
                "title": "Existing Docs",
            }
        }
        result = _generate_toctree_entries(sources)
        assert "existing/index" in result
        assert SOURCES_STAGING_DIR not in result

    def test_source_without_title(self):
        sources = {
            "transform": {
                "artifact_path": "target/artifacts/transform",
            }
        }
        result = _generate_toctree_entries(sources)
        assert ":caption: transform" in result


class TestStageSources(object):
    def test_copies_docs_from_artifact_path(self, tmp_path):
        repo_path = tmp_path / "repo"
        docs_path = repo_path / "docs"
        docs_path.mkdir(parents=True)

        artifact_dir = repo_path / "target" / "artifacts" / "transform" / "docs"
        artifact_dir.mkdir(parents=True)
        (artifact_dir / "index.rst").write_text("Transform Index")
        (artifact_dir / "api.rst").write_text("API Content")

        sources = {
            "transform": {
                "artifact_path": "target/artifacts/transform",
                "docs_root": "docs",
                "title": "Transform Service API",
            }
        }

        staged = _stage_sources(repo_path, docs_path, sources)
        staging_dir = docs_path / SOURCES_STAGING_DIR / "transform"
        assert staging_dir.exists()
        assert (staging_dir / "index.rst").read_text() == "Transform Index"
        assert (staging_dir / "api.rst").read_text() == "API Content"
        assert len(staged) == 1

    def test_creates_staging_dir(self, tmp_path):
        repo_path = tmp_path / "repo"
        docs_path = repo_path / "docs"
        docs_path.mkdir(parents=True)

        artifact_dir = repo_path / "target" / "artifacts" / "svc" / "docs"
        artifact_dir.mkdir(parents=True)
        (artifact_dir / "index.rst").write_text("content")

        sources = {
            "svc": {
                "artifact_path": "target/artifacts/svc",
                "docs_root": "docs",
                "title": "Service",
            }
        }

        _stage_sources(repo_path, docs_path, sources)
        assert (docs_path / SOURCES_STAGING_DIR).is_dir()

    def test_warns_on_missing_artifact(self, tmp_path, capsys):
        repo_path = tmp_path / "repo"
        docs_path = repo_path / "docs"
        docs_path.mkdir(parents=True)

        sources = {
            "missing": {
                "artifact_path": "target/artifacts/missing",
                "docs_root": "docs",
                "title": "Missing Service",
            }
        }

        staged = _stage_sources(repo_path, docs_path, sources)
        assert len(staged) == 0
        captured = capsys.readouterr()
        assert "Warning" in captured.out

    def test_default_docs_root(self, tmp_path):
        repo_path = tmp_path / "repo"
        docs_path = repo_path / "docs"
        docs_path.mkdir(parents=True)

        artifact_dir = repo_path / "target" / "artifacts" / "svc" / "docs"
        artifact_dir.mkdir(parents=True)
        (artifact_dir / "index.rst").write_text("content")

        sources = {
            "svc": {
                "artifact_path": "target/artifacts/svc",
                "title": "Service",
            }
        }

        staged = _stage_sources(repo_path, docs_path, sources)
        assert len(staged) == 1
        assert (docs_path / SOURCES_STAGING_DIR / "svc" / "index.rst").exists()

    def test_skips_sources_without_artifact_path(self, tmp_path):
        repo_path = tmp_path / "repo"
        docs_path = repo_path / "docs"
        docs_path.mkdir(parents=True)

        sources = {
            "existing": {
                "title": "Existing Docs",
            }
        }

        staged = _stage_sources(repo_path, docs_path, sources)
        assert len(staged) == 0


class TestInjectSources:
    def test_marker_replacement(self, tmp_path):
        index_path = tmp_path / "index.rst"
        original = textwrap.dedent("""\
            Welcome
            =======

            .. bartleby-sources::

            Indices and tables
            ==================
        """)
        index_path.write_text(original)

        sources = {
            "transform": {
                "artifact_path": "target/artifacts/transform",
                "title": "Transform API",
            }
        }

        saved = _inject_sources(index_path, sources)
        assert saved == original
        modified = index_path.read_text()
        assert ".. toctree::" in modified
        assert SOURCES_MARKER not in modified
        assert ":caption: Transform API" in modified

    def test_indexes_and_tables_fallback(self, tmp_path):
        index_path = tmp_path / "index.rst"
        original = textwrap.dedent("""\
            Welcome
            =======

            Some content here.

            Indexes and tables
            ==================
        """)
        index_path.write_text(original)

        sources = {
            "svc": {
                "artifact_path": "target/artifacts/svc",
                "title": "Service",
            }
        }

        saved = _inject_sources(index_path, sources)
        assert saved == original
        modified = index_path.read_text()
        assert ".. toctree::" in modified
        assert modified.index(".. toctree::") < modified.index("Indexes and tables")

    def test_indices_and_tables_fallback(self, tmp_path):
        index_path = tmp_path / "index.rst"
        original = textwrap.dedent("""\
            Welcome
            =======

            Indices and tables
            ==================
        """)
        index_path.write_text(original)

        sources = {
            "svc": {
                "artifact_path": "target/artifacts/svc",
                "title": "Service",
            }
        }

        _inject_sources(index_path, sources)
        modified = index_path.read_text()
        assert ".. toctree::" in modified
        assert modified.index(".. toctree::") < modified.index("Indices and tables")

    def test_append_fallback(self, tmp_path):
        index_path = tmp_path / "index.rst"
        original = textwrap.dedent("""\
            Welcome
            =======

            Some content.
        """)
        index_path.write_text(original)

        sources = {
            "svc": {
                "artifact_path": "target/artifacts/svc",
                "title": "Service",
            }
        }

        _inject_sources(index_path, sources)
        modified = index_path.read_text()
        assert modified.endswith("\n")
        assert ".. toctree::" in modified

    def test_no_sources_unchanged(self, tmp_path):
        index_path = tmp_path / "index.rst"
        original = "Welcome\n=======\n"
        index_path.write_text(original)

        saved = _inject_sources(index_path, {})
        assert saved is None
        assert index_path.read_text() == original


class TestRestoreIndex:
    def test_restores_original_content(self, tmp_path):
        index_path = tmp_path / "index.rst"
        original = "Original content\n"
        index_path.write_text("Modified content\n")

        _restore_index(index_path, original)
        assert index_path.read_text() == original


class TestValidateSourcePaths:
    def test_all_exist(self, tmp_path):
        repo_path = tmp_path / "repo"
        docs_path = repo_path / "docs"
        docs_path.mkdir(parents=True)

        artifact_dir = repo_path / "target" / "artifacts" / "svc" / "docs"
        artifact_dir.mkdir(parents=True)

        sources = {
            "svc": {
                "artifact_path": "target/artifacts/svc",
                "docs_root": "docs",
                "title": "Service",
            }
        }

        result = _validate_source_paths(repo_path, docs_path, sources)
        assert "svc" in result

    def test_some_missing(self, tmp_path, capsys):
        repo_path = tmp_path / "repo"
        docs_path = repo_path / "docs"
        docs_path.mkdir(parents=True)

        artifact_dir = repo_path / "target" / "artifacts" / "svc" / "docs"
        artifact_dir.mkdir(parents=True)

        sources = {
            "svc": {
                "artifact_path": "target/artifacts/svc",
                "docs_root": "docs",
                "title": "Service",
            },
            "missing": {
                "artifact_path": "target/artifacts/missing",
                "docs_root": "docs",
                "title": "Missing",
            },
        }

        result = _validate_source_paths(repo_path, docs_path, sources)
        assert "svc" in result
        assert "missing" not in result
        captured = capsys.readouterr()
        assert "Warning" in captured.out

    def test_empty_sources(self, tmp_path):
        repo_path = tmp_path / "repo"
        docs_path = repo_path / "docs"
        docs_path.mkdir(parents=True)

        result = _validate_source_paths(repo_path, docs_path, {})
        assert result == {}

    def test_source_without_artifact_path_checks_docs(self, tmp_path):
        repo_path = tmp_path / "repo"
        docs_path = repo_path / "docs"
        (docs_path / "existing").mkdir(parents=True)

        sources = {
            "existing": {
                "title": "Existing Docs",
            }
        }

        result = _validate_source_paths(repo_path, docs_path, sources)
        assert "existing" in result

    def test_source_without_artifact_path_missing_in_docs(self, tmp_path, capsys):
        repo_path = tmp_path / "repo"
        docs_path = repo_path / "docs"
        docs_path.mkdir(parents=True)

        sources = {
            "nowhere": {
                "title": "Nowhere Docs",
            }
        }

        result = _validate_source_paths(repo_path, docs_path, sources)
        assert "nowhere" not in result
        captured = capsys.readouterr()
        assert "Warning" in captured.out


class TestCleanupStagedSources:
    def test_removes_staging_dir(self, tmp_path):
        staging = tmp_path / SOURCES_STAGING_DIR
        staging.mkdir()
        (staging / "transform").mkdir()
        (staging / "transform" / "index.rst").write_text("content")

        _cleanup_staged_sources(tmp_path)
        assert not staging.exists()

    def test_no_error_if_missing(self, tmp_path):
        _cleanup_staged_sources(tmp_path)


class TestRunBuilds:
    def _make_controller(self):
        ctrl = object.__new__(LocalController)
        ctrl.app = MagicMock()
        ctrl.app.pargs.shell = "all"
        ctrl.app.pargs.root_doc = "all"
        ctrl.app.pargs.autodoc = False
        ctrl.app.pargs.gather = ""
        ctrl.app.pargs.repo_name = "test"
        ctrl.app.pargs.repo_version = "1.0"
        ctrl.app.pargs.confidential = False
        ctrl.app.pargs.default_logo = None
        ctrl.app.pargs.html_default_logo = None
        ctrl.app.pargs.pdf_default_logo = None
        ctrl.app.pargs.document_title = None
        ctrl.app.pargs.timestamp_title = False
        return ctrl

    @patch("hmd_cli_bartleby.controller.read_manifest", return_value={})
    @patch.object(LocalController, "_run_transform")
    def test_index_restored_after_success(
        self, mock_transform, mock_manifest, tmp_path
    ):
        ctrl = self._make_controller()
        docs_path = tmp_path / "docs"
        docs_path.mkdir()
        index_path = docs_path / "index.rst"
        original = "Original\n========\n\n.. bartleby-sources::\n"
        index_path.write_text(original)

        sources = {
            "svc": {
                "artifact_path": "target/artifacts/svc",
                "title": "Service",
            }
        }

        manifest = {"bartleby": {"sources": sources}}

        with patch("hmd_cli_bartleby.controller.read_manifest", return_value=manifest):
            with patch("os.getcwd", return_value=str(tmp_path)):
                artifact_docs = tmp_path / "target" / "artifacts" / "svc" / "docs"
                artifact_docs.mkdir(parents=True)
                (artifact_docs / "index.rst").write_text("Svc docs")

                builds = [
                    {
                        "name": "index",
                        "shell": "html",
                        "root_doc": "index",
                        "config": {},
                    }
                ]
                ctrl._run_builds(builds)

        assert index_path.read_text() == original
        assert not (docs_path / SOURCES_STAGING_DIR).exists()

    @patch("hmd_cli_bartleby.controller.read_manifest", return_value={})
    @patch.object(
        LocalController, "_run_transform", side_effect=RuntimeError("build failed")
    )
    def test_index_restored_on_exception(self, mock_transform, mock_manifest, tmp_path):
        ctrl = self._make_controller()
        docs_path = tmp_path / "docs"
        docs_path.mkdir()
        index_path = docs_path / "index.rst"
        original = "Original\n========\n\n.. bartleby-sources::\n"
        index_path.write_text(original)

        sources = {
            "svc": {
                "artifact_path": "target/artifacts/svc",
                "title": "Service",
            }
        }

        manifest = {"bartleby": {"sources": sources}}

        with patch("hmd_cli_bartleby.controller.read_manifest", return_value=manifest):
            with patch("os.getcwd", return_value=str(tmp_path)):
                artifact_docs = tmp_path / "target" / "artifacts" / "svc" / "docs"
                artifact_docs.mkdir(parents=True)
                (artifact_docs / "index.rst").write_text("Svc docs")

                builds = [
                    {
                        "name": "index",
                        "shell": "html",
                        "root_doc": "index",
                        "config": {},
                    }
                ]
                with pytest.raises(RuntimeError):
                    ctrl._run_builds(builds)

        assert index_path.read_text() == original
        assert not (docs_path / SOURCES_STAGING_DIR).exists()

    @patch("hmd_cli_bartleby.controller.read_manifest", return_value={})
    @patch.object(LocalController, "_run_transform")
    def test_staging_dir_cleaned_up(self, mock_transform, mock_manifest, tmp_path):
        ctrl = self._make_controller()
        docs_path = tmp_path / "docs"
        docs_path.mkdir()
        index_path = docs_path / "index.rst"
        index_path.write_text("Title\n=====\n")

        sources = {
            "svc": {
                "artifact_path": "target/artifacts/svc",
                "title": "Service",
            }
        }

        manifest = {"bartleby": {"sources": sources}}

        with patch("hmd_cli_bartleby.controller.read_manifest", return_value=manifest):
            with patch("os.getcwd", return_value=str(tmp_path)):
                artifact_docs = tmp_path / "target" / "artifacts" / "svc" / "docs"
                artifact_docs.mkdir(parents=True)
                (artifact_docs / "index.rst").write_text("Svc docs")

                builds = [
                    {
                        "name": "index",
                        "shell": "html",
                        "root_doc": "index",
                        "config": {},
                    }
                ]
                ctrl._run_builds(builds)

        assert not (docs_path / SOURCES_STAGING_DIR).exists()

    @patch("hmd_cli_bartleby.controller.read_manifest", return_value={})
    @patch.object(LocalController, "_run_transform")
    def test_no_sources_runs_transforms_directly(
        self, mock_transform, mock_manifest, tmp_path
    ):
        ctrl = self._make_controller()

        with patch("os.getcwd", return_value=str(tmp_path)):
            docs_path = tmp_path / "docs"
            docs_path.mkdir()
            (docs_path / "index.rst").write_text("Title\n=====\n")

            builds = [
                {"name": "index", "shell": "html", "root_doc": "index", "config": {}}
            ]
            ctrl._run_builds(builds)

        mock_transform.assert_called_once()
