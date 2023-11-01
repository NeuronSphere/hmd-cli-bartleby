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
                    "help": "The root docuemnt to pass to the bartleby transform instance.",
                    "default": "all",
                },
            ),
            *[param["arg"] for _, param in BARTLEBY_PARAMETERS.items()],
        )

    def _default(self):
        """Default action if no sub-command is passed."""
        load_hmd_env(override=False)
        shell = self.app.pargs.shell
        root_doc = self.app.pargs.root_doc

        docs = self._get_documents(root_doc=root_doc, shell=shell)
        builds = self._get_shells(docs, shell=shell)

        for build in builds:
            self._run_transform(
                build["name"], build["shell"], build["root_doc"], build["config"]
            )

    def _get_documents(self, root_doc: str = "all", shell: str = "all"):
        manifest = read_manifest()
        roots = manifest.get("bartleby", {}).get("roots")

        if roots is None:
            return {"index": {"builders": [shell], "root_doc": "index"}}
        docs = {}

        if root_doc == "all":
            return roots
        else:
            docs[root_doc] = roots.get(root_doc, {})
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
        docs = self._get_documents(shell="html")
        builds = self._get_shells(docs, shell="html")

        for build in builds:
            self._run_transform(
                build["name"], build["shell"], build["root_doc"], build["config"]
            )

    @ex(help="Render PDF documentation", arguments=[])
    def pdf(self):
        load_hmd_env(override=False)
        docs = self._get_documents(shell="pdf")
        builds = self._get_shells(docs, shell="pdf")

        for build in builds:
            self._run_transform(
                build["name"], build["shell"], build["root_doc"], build["config"]
            )

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
