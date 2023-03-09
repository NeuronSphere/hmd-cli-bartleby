import os
import shutil
from importlib.metadata import version
from cement import Controller, ex
from pathlib import Path
from glob import glob
from hmd_cli_tools.hmd_cli_tools import cd, load_hmd_env, set_hmd_env
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
    },
    "HMD_BARTLEBY_HTML_DEFAULT_LOGO": {
        "hidden": True,
        "default": "https://neuronsphere.io/hubfs/bartleby_assets/Neuron_Sphere_Logo_RGB_Color.png",
    },
    "HMD_BARTLEBY_PDF_DEFAULT_LOGO": {
        "hidden": True,
        "default": "https://neuronsphere.io/hubfs/bartleby_assets/NeuronSphereSwoosh.jpg",
    },
}


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
                    "default": "default",
                },
            ),
        )

    def _default(self):
        """Default action if no sub-command is passed."""
        load_hmd_env(override=False)

        args = {}
        name = self.app.pargs.repo_name
        repo_version = self.app.pargs.repo_version

        autodoc = self.app.pargs.autodoc
        gather = self.app.pargs.gather
        shell = self.app.pargs.shell

        if len(gather) > 0:
            gather_repos(gather)
            args.update({"gather": gather})

        image_name = f"{os.environ.get('HMD_CONTAINER_REGISTRY', 'ghcr.io/neuronsphere')}/hmd-tf-bartleby:{os.environ.get('HMD_TF_BARTLEBY_VERSION', 'stable')}"
        print(image_name)
        if len(shell.split(",")) > 1:
            for cmd in shell.split(","):
                transform_instance_context = {"shell": f"{cmd.strip()}"}
                args.update(
                    {
                        "name": name,
                        "version": repo_version,
                        "transform_instance_context": transform_instance_context,
                        "image_name": image_name,
                        "autodoc": autodoc,
                    }
                )

                from .hmd_cli_bartleby import transform

                transform(**args)
        else:
            transform_instance_context = {"shell": shell}

            args.update(
                {
                    "name": name,
                    "version": repo_version,
                    "transform_instance_context": transform_instance_context,
                    "image_name": image_name,
                    "autodoc": autodoc,
                }
            )

            from .hmd_cli_bartleby import transform

            transform(**args)

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
