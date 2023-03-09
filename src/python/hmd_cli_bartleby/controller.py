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


def _get_parameter_default(param: str, manifest: Dict, default: Any = None):
    value = manifest.get(BARTLEBY_PARAMETERS[param]["key"])

    if value is None:
        value = get_env_var(
            BARTLEBY_PARAMETERS[param]["env_var"], throw=False, default=default
        )

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
                    "default": "html,pdf",
                },
            ),
            *[param["arg"] for _, param in BARTLEBY_PARAMETERS.items()],
        )

    def _default(self):
        """Default action if no sub-command is passed."""
        load_hmd_env(override=False)
        shell = self.app.pargs.shell

        if len(shell.split(",")) > 1:
            for cmd in shell.split(","):
                self._run_transform(f"{cmd.strip()}")
        else:
            self._run_transform(shell)

    def _run_transform(self, shell: str):

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

        transform_instance_context = {"shell": shell}

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
            }
        )

        from .hmd_cli_bartleby import transform

        transform(**args)

    @ex(help="Render HTML documentation", arguments=[])
    def html(self):
        load_hmd_env(override=False)
        self._run_transform("html")

    @ex(help="Render PDF documentation", arguments=[])
    def pdf(self):
        load_hmd_env(override=False)
        self._run_transform("pdf")

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
