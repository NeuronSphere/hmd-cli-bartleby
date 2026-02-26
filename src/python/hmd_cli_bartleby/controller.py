import json
import os
import shutil
from importlib.metadata import version
from typing import Any, Dict
from cement import Controller, ex
from pathlib import Path
from glob import glob
from hmd_cli_tools.hmd_cli_tools import (
    cd,
    load_hmd_env,
    set_hmd_env,
    get_env_var,
    read_manifest,
)
from hmd_cli_tools.prompt_tools import prompt_for_values

VERSION_BANNER = """
hmd bartleby version: {}
"""

repo_types = {
    "app": {"name": "Applications"},
    "cli": {"name": "Commands"},
    "client": {"name": "Clients"},
    "config": {"name": "Configurations"},
    "dbt": {"name": "DBT_Transforms"},
    "docs": {"name": "Documentation"},
    "inf": {"name": "Infrastructure"},
    "inst": {"name": "Instances"},
    "installer": {"name": "Installer"},
    "img": {"name": "Docker_Images"},
    "lang": {"name": "Language_Packs"},
    "lib": {"name": "Libraries"},
    "ms": {"name": "Microservices"},
    "orb": {"name": "CircleCI_Orbs"},
    "tf": {"name": "Transforms"},
    "ui": {"name": "UI_Components"},
}

DEFAULT_CONFIG = {
    "HMD_BARTLEBY_DEFAULT_LOGO": {
        "hidden": True,
        "default": "https://neuronsphere.io/hubfs/bartleby_assets/NeuronSphereSwoosh.jpg",
    }
}

DEFAULT_BUILDERS = ["html", "pdf"]

SOURCES_MARKER = ".. bartleby-sources::"
INDEXES_MARKERS = ["Indexes and tables\n", "Indices and tables\n"]
SOURCES_STAGING_DIR = "_sources"


BARTLEBY_PARAMETERS = {
    "document_title": {
        "arg": (
            ["--title"],
            {
                "action": "store",
                "dest": "document_title",
                "help": "specify document title",
            },
        )
    },
    "timestamp_title": {
        "arg": (
            ["--no-timestamp-title"],
            {
                "action": "store_true",
                "dest": "timestamp_title",
                "help": "append timestamp to title",
                "default": False,
            },
        )
    },
    "confidential": {
        "arg": (
            ["--confidential"],
            {
                "action": "store_true",
                "dest": "confidential",
                "help": "The flag to indicate if documents should include HMD_BARTLEBY_CONFIDENTIALITY_STATEMENT",
                "default": False,
            },
        ),
        "key": "confidential",
        "env_var": "HMD_BARTLEBY_CONFIDENTIAL",
    },
    "default_logo": {
        "arg": (
            ["--default-logo"],
            {
                "action": "store",
                "dest": "default_logo",
                "help": "URL to default HTML logo or PDF cover image to use",
            },
        ),
        "key": "default_logo",
        "env_var": "HMD_BARTLEBY_DEFAULT_LOGO",
    },
    "html_default_logo": {
        "arg": (
            ["--html-default-logo"],
            {
                "action": "store",
                "dest": "html_default_logo",
                "help": "URL to default HTML logo",
            },
        ),
        "key": "default_logo",
        "env_var": "HMD_BARTLEBY_HTML_DEFAULT_LOGO",
    },
    "pdf_default_logo": {
        "arg": (
            ["--pdf-default-logo"],
            {
                "action": "store",
                "dest": "pdf_default_logo",
                "help": "URL to default PDF cover image to use",
            },
        ),
        "key": "default_logo",
        "env_var": "HMD_BARTLEBY_PDF_DEFAULT_LOGO",
    },
}


def _get_default_builder_config(shell: str):
    prefix = f"HMD_BARTLEBY__{shell.upper()}__"
    config = {}
    for key, value in os.environ.items():
        if key.startswith(prefix):
            config_var = key.removeprefix(prefix).lower()
            config[config_var] = value

    return config


def _get_parameter_default(param: str, manifest: Dict, default: Any = None):
    bartleby_param = BARTLEBY_PARAMETERS.get(param)
    bartleby_manifest = manifest.get("bartleby", {}).get("config", {})

    if bartleby_param is None:
        value = bartleby_manifest.get("builders", {}).get(param)

        if value is None:
            value = json.loads(
                get_env_var(
                    f"HMD_BARTLEBY_{param.upper()}_CONFIG",
                    throw=False,
                    default=str(default),
                )
            )

        return value

    value = bartleby_manifest.get(bartleby_param["key"])

    if value is None:
        value = get_env_var(bartleby_param["env_var"], throw=False, default=default)

    return value


def _get_sources(manifest: Dict) -> Dict:
    return manifest.get("bartleby", {}).get("sources", {})


def _stage_sources(repo_path: Path, docs_path: Path, sources: Dict) -> "list[Path]":
    staged = []
    for key, source in sources.items():
        artifact_path = source.get("artifact_path")
        if artifact_path is None:
            continue
        docs_root = source.get("docs_root", "docs")
        src_dir = repo_path / artifact_path / docs_root
        if not src_dir.exists():
            print(
                f"Warning: artifact docs path '{src_dir}' does not exist for "
                f"source '{key}'. Skipping."
            )
            continue
        dest_dir = docs_path / SOURCES_STAGING_DIR / key
        if dest_dir.exists():
            shutil.rmtree(dest_dir)
        shutil.copytree(src_dir, dest_dir)
        staged.append(dest_dir)
    return staged


def _cleanup_staged_sources(docs_path: Path):
    staging = docs_path / SOURCES_STAGING_DIR
    if staging.exists():
        shutil.rmtree(staging)


def _generate_toctree_entries(sources: Dict) -> str:
    if not sources:
        return ""
    blocks = []
    for key, source in sources.items():
        title = source.get("title", key)
        if source.get("artifact_path"):
            path = f"{SOURCES_STAGING_DIR}/{key}/index"
        else:
            path = f"{key}/index"
        block = (
            f".. toctree::\n"
            f"   :maxdepth: 2\n"
            f"   :caption: {title}\n"
            f"\n"
            f"   {path}\n"
        )
        blocks.append(block)
    return "\n".join(blocks) + "\n"


def _inject_sources(index_path: Path, sources: Dict) -> "str | None":
    if not sources:
        return None
    original = index_path.read_text()
    toctree = _generate_toctree_entries(sources)
    lines = original.splitlines(keepends=True)

    # Strategy 1: marker replacement
    marker_idx = None
    for i, line in enumerate(lines):
        if SOURCES_MARKER in line:
            marker_idx = i
            break

    if marker_idx is not None:
        lines[marker_idx] = toctree
        index_path.write_text("".join(lines))
        return original

    # Strategy 2: insert before "Indexes and tables" or "Indices and tables"
    for i, line in enumerate(lines):
        if line in INDEXES_MARKERS:
            lines.insert(i, toctree + "\n")
            index_path.write_text("".join(lines))
            return original

    # Strategy 3: append
    text = original
    if not text.endswith("\n"):
        text += "\n"
    text += "\n" + toctree
    index_path.write_text(text)
    return original


def _restore_index(index_path: Path, original_content: str):
    index_path.write_text(original_content)


def _validate_source_paths(repo_path: Path, docs_path: Path, sources: Dict) -> Dict:
    valid = {}
    for key, source in sources.items():
        artifact_path = source.get("artifact_path")
        if artifact_path:
            docs_root = source.get("docs_root", "docs")
            check_path = repo_path / artifact_path / docs_root
            if check_path.exists():
                valid[key] = source
            else:
                print(
                    f"Warning: source '{key}' artifact docs path "
                    f"'{check_path}' not found. Skipping."
                )
        else:
            check_path = docs_path / key
            if check_path.exists():
                valid[key] = source
            else:
                print(
                    f"Warning: source '{key}' docs path "
                    f"'{check_path}' not found. Skipping."
                )
    return valid


def update_index(index_path, repo):
    with open(index_path, "r") as index:
        text = index.readlines()
        i = [text.index(x) for x in text if x == "Indexes and tables\n"][0]
        text.insert(i, f"   {repo}/index.rst\n")
    with open(index_path, "w") as index:
        index.writelines(text)


def gather_repos(gather):
    path_cwd = Path(os.getcwd())
    customer_code = os.environ.get("HMD_CUSTOMER_CODE", "hmd")
    if os.path.basename(
        path_cwd
    ) == "hmd-docs-bartleby" and "hmd-lib-bartleby-demos" in os.listdir(
        path_cwd.parent
    ):
        docs_path = path_cwd / "docs"
        for dirs in [dirs for dirs in os.listdir(docs_path) if dirs != "index.rst"]:
            shutil.rmtree(docs_path / dirs)
        index_path = path_cwd.parent / "hmd-lib-bartleby-demos" / "docs" / "index.rst"
        if index_path.exists():
            shutil.copyfile(index_path, docs_path / "index.rst")
        else:
            raise Exception(f"Path {index_path} does not exist.")
        gather = gather.split(",")
        for repo in gather:
            if len(repo.split("-")) > 1:
                repo_path = path_cwd.parent / repo
                if repo_path.exists() and "docs" in os.listdir(repo_path):
                    shutil.copytree(repo_path / "docs", docs_path / repo)
                else:
                    raise Exception(
                        f"Repository {repo} docs folder could not be located. Ensure the repo is "
                        f"available with a docs folder in the parent directory of the current path."
                    )
                update_index(docs_path / "index.rst", repo)

    else:
        raise Exception(
            "Gather mode can only be used from the bartleby docs repo (hmd-docs-bartleby) and the"
            "bartleby library (hmd-lib-bartleby-demos) must be available in the parent directory"
            "of the current path."
        )


class LocalController(Controller):
    class Meta:
        label = "bartleby"

        stacked_type = "nested"
        stacked_on = "base"

        description = "Run bartleby transforms to generate rendered documents"

        arguments = (
            (
                ["-v", "--version"],
                {
                    "help": "Display the version of the bartleby command.",
                    "action": "version",
                    "version": VERSION_BANNER.format(version("hmd_cli_bartleby")),
                },
            ),
            (
                ["-a", "--autodoc"],
                {
                    "action": "store_true",
                    "dest": "autodoc",
                    "help": "The flag to indicate if python modules exist and should be added to the autosummary.",
                    "default": False,
                },
            ),
            (
                ["-g", "--gather"],
                {
                    "action": "store",
                    "dest": "gather",
                    "help": "The list of repositories or repository types to transform.",
                    "default": "",
                },
            ),
            (
                ["-s", "--shell"],
                {
                    "action": "store",
                    "dest": "shell",
                    "help": "The command to pass to the bartleby transform instance.",
                    "default": "all",
                },
            ),
            (
                ["-rd", "--root-doc"],
                {
                    "action": "store",
                    "dest": "root_doc",
                    "help": "Root document(s) to build. Use 'all' for all roots, or specify "
                    "comma-separated root names (e.g., 'guide,api').",
                    "default": "all",
                },
            ),
            *[param["arg"] for _, param in BARTLEBY_PARAMETERS.items()],
        )

    def _run_builds(self, builds):
        repo_path = Path(os.getcwd())
        docs_path = repo_path / "docs"
        manifest = read_manifest()
        sources = _get_sources(manifest)

        if not sources:
            for build in builds:
                self._run_transform(
                    build["name"], build["shell"], build["root_doc"], build["config"]
                )
            return

        valid_sources = _validate_source_paths(repo_path, docs_path, sources)
        if not valid_sources:
            for build in builds:
                self._run_transform(
                    build["name"], build["shell"], build["root_doc"], build["config"]
                )
            return

        _stage_sources(repo_path, docs_path, valid_sources)

        root_docs = {b["root_doc"] for b in builds}
        originals = {}
        for root_doc in root_docs:
            index_path = docs_path / f"{root_doc}.rst"
            if index_path.exists():
                original = _inject_sources(index_path, valid_sources)
                if original is not None:
                    originals[index_path] = original

        try:
            for build in builds:
                self._run_transform(
                    build["name"], build["shell"], build["root_doc"], build["config"]
                )
        finally:
            for index_path, original in originals.items():
                _restore_index(index_path, original)
            _cleanup_staged_sources(docs_path)

    def _default(self):
        """Default action if no sub-command is passed."""
        load_hmd_env(override=False)
        shell = self.app.pargs.shell
        root_doc = self.app.pargs.root_doc

        docs = self._get_documents(root_doc=root_doc, shell=shell)
        builds = self._get_shells(docs, shell=shell)

        self._run_builds(builds)

    def _get_documents(self, root_doc: str = "all", shell: str = "all"):
        manifest = read_manifest()
        roots = manifest.get("bartleby", {}).get("roots")

        if roots is None:
            return {"index": {"builders": DEFAULT_BUILDERS, "root_doc": "index"}}

        if root_doc == "all":
            return roots

        requested = [name.strip() for name in root_doc.split(",") if name.strip()]
        docs = {}
        for name in requested:
            if name in roots:
                docs[name] = roots[name]
            else:
                print(
                    f"Warning: root document '{name}' not found in manifest. "
                    f"Available roots: {', '.join(roots.keys())}"
                )

        if not docs:
            raise SystemExit(
                f"Error: none of the requested root documents ({root_doc}) "
                f"were found in manifest. Available roots: {', '.join(roots.keys())}"
            )

        return docs

    def _get_shells(self, docs: dict, shell: str = "all"):
        tf_ctxs = []
        manifest = read_manifest()

        for root, doc in docs.items():
            shells = doc.get("builders", [])
            doc_config = doc.get("config", {})
            for s in shells:
                if isinstance(s, dict):
                    s = s.get("shell")
                    config = s.get("config", _get_parameter_default(s, manifest, {}))
                else:
                    config = _get_parameter_default(s, manifest, {})

                env_config = _get_default_builder_config(s)
                config = {**doc_config, **config, **env_config}

                if shell == "all" or s == shell:
                    tf_ctxs.append(
                        {
                            "name": root,
                            "shell": s,
                            "root_doc": doc.get("root_doc", "index"),
                            "config": config,
                        }
                    )

        return tf_ctxs

    def _run_transform(self, doc_name: str, shell: str, root_doc: str, config: dict):
        args = {}
        name = self.app.pargs.repo_name
        repo_version = self.app.pargs.repo_version

        autodoc = self.app.pargs.autodoc
        gather = self.app.pargs.gather

        manifest = read_manifest()
        confidential = _get_parameter_default(
            "confidential", manifest, default=self.app.pargs.confidential
        )

        default_logo = self.app.pargs.default_logo

        if self.app.pargs.default_logo is None:
            default_logo = _get_parameter_default(
                "default_logo", manifest, self.app.pargs.default_logo
            )

        html_default_logo = self.app.pargs.html_default_logo
        if html_default_logo is None:
            html_default_logo = _get_parameter_default(
                "html_default_logo", manifest, default_logo
            )

        pdf_default_logo = self.app.pargs.pdf_default_logo
        if pdf_default_logo is None:
            pdf_default_logo = _get_parameter_default(
                "pdf_default_logo", manifest, default_logo
            )

        if len(gather) > 0:
            gather_repos(gather)
            args.update({"gather": gather})

        image_name = f"{os.environ.get('HMD_CONTAINER_REGISTRY', 'ghcr.io/neuronsphere')}/hmd-tf-bartleby:{os.environ.get('HMD_TF_BARTLEBY_VERSION', 'stable')}"

        transform_instance_context = {
            "name": doc_name,
            "shell": shell,
            "root_doc": root_doc,
            "config": config,
        }

        args.update(
            {
                "name": name,
                "version": repo_version,
                "transform_instance_context": transform_instance_context,
                "image_name": image_name,
                "autodoc": autodoc,
                "confidential": confidential,
                "default_logo": default_logo,
                "html_default_logo": html_default_logo,
                "pdf_default_logo": pdf_default_logo,
                "document_title": self.app.pargs.document_title,
                "timestamp_title": self.app.pargs.timestamp_title,
            }
        )

        from .hmd_cli_bartleby import transform

        transform(**args)

    @ex(help="Render HTML documentation", arguments=[])
    def html(self):
        load_hmd_env(override=False)
        root_doc = self.app.pargs.root_doc
        docs = self._get_documents(root_doc=root_doc, shell="html")
        builds = self._get_shells(docs, shell="html")
        self._run_builds(builds)

    @ex(help="Render PDF documentation", arguments=[])
    def pdf(self):
        load_hmd_env(override=False)
        root_doc = self.app.pargs.root_doc
        docs = self._get_documents(root_doc=root_doc, shell="pdf")
        builds = self._get_shells(docs, shell="pdf")
        self._run_builds(builds)

    @ex(help="Render RevealJS slideshow", arguments=[])
    def slides(self):
        load_hmd_env(override=False)
        root_doc = self.app.pargs.root_doc
        docs = self._get_documents(root_doc=root_doc, shell="revealjs")
        builds = self._get_shells(docs, shell="revealjs")
        self._run_builds(builds)

    @ex(help="Render images from puml", arguments=[])
    def puml(self):
        load_hmd_env(override=False)

        def get_files():
            files = glob("**", recursive=True)
            files = list(map(lambda x: x.replace("\\", "/"), files))
            return files

        input_path = Path(os.getcwd()) / "docs"
        output_path = Path(os.getcwd()) / "target" / "bartleby" / "puml_images"
        image_name = f"{os.environ.get('HMD_CONTAINER_REGISTRY', 'ghcr.io/neuronsphere')}/hmd-tf-bartleby:{os.environ.get('HMD_TF_BARTLEBY_VERSION', 'stable')}"

        if not output_path.exists():
            os.makedirs(output_path)
        if input_path.exists():
            with cd(input_path):
                puml_files = list(filter(lambda x: (x.endswith(".puml")), get_files()))
                if len(puml_files) > 0:
                    from .hmd_cli_bartleby import transform_puml

                    transform_puml(puml_files, input_path, output_path, image_name)
                else:
                    print(
                        "No puml files found in the docs folder of the current directory."
                    )

    @ex(help="Configure Bartleby environment variables", arguments=[])
    def configure(self):
        load_hmd_env()
        results = prompt_for_values(DEFAULT_CONFIG)

        if results:
            for k, v in results.items():
                set_hmd_env(k, str(v))

    @ex(help="Pull the latest Bartleby image", arguments=[])
    def update_image(self):
        load_hmd_env()
        from .hmd_cli_bartleby import update_image as do_update_image

        image_name = f"{os.environ.get('HMD_CONTAINER_REGISTRY', 'ghcr.io/neuronsphere')}/hmd-tf-bartleby"
        img_tag = os.environ.get("HMD_TF_BARTLEBY_VERSION", "stable")

        do_update_image(image_name=f"{image_name}:{img_tag}")
