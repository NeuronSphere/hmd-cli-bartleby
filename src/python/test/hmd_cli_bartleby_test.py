from unittest.mock import patch
from hmd_cli_bartleby.controller import LocalController, DEFAULT_BUILDERS


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
