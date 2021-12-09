import os
from pathlib import Path
from cement.utils.shell import exec_cmd2
from typing import List, Dict
from hmd_cli_tools.hmd_cli_tools import get_version, cd


hmd_repo_home = os.environ["HMD_REPO_HOME"]
hmd_home = os.environ["HMD_HOME"]


def transform(
    name: str or List[str],
    version: str,
    transform_instance_context: Dict,
    image_name: str,
    autodoc: bool = False,
):
    if hmd_repo_home:
        repo_path = Path(hmd_repo_home) / name
    else:
        repo_path = Path(os.getcwd()) / name

    if not repo_path.exists():
        raise Exception("Repository root could not be located.")

    with cd(repo_path):
        bartleby_version = get_version()

    input_path = repo_path / "docs"
    output_path = repo_path / "target" / "bartleby"

    try:
        if Path(repo_path / "src" / "python").exists() and autodoc:
            code_path = repo_path / "src" / "python"

            command = [
                "docker",
                "run",
                "--rm",
                "-v",
                f"{input_path}:/hmd_transform/input",
                "-v",
                f"{output_path}:/hmd_transform/output",
                "-v",
                f"{code_path}:/code/src/python",
                "-e",
                f"TRANSFORM_INSTANCE_CONTEXT={transform_instance_context}",
                "-e",
                f"HMD_DOC_REPO_NAME={name}",
                "-e",
                f"HMD_DOC_REPO_VERSION={version}",
                "-e",
                f"AUTODOC=True",
                f"{image_name}:{bartleby_version}",
            ]

            return_code = exec_cmd2(command)
        else:
            command = [
                "docker",
                "run",
                "--rm",
                "-v",
                f"{input_path}:/hmd_transform/input",
                "-v",
                f"{output_path}:/hmd_transform/output",
                "-e",
                f"TRANSFORM_INSTANCE_CONTEXT={transform_instance_context}",
                "-e",
                f"HMD_DOC_REPO_NAME={name}",
                "-e",
                f"HMD_DOC_REPO_VERSION={version}",
                f"{image_name}:{bartleby_version}",
            ]

            return_code = exec_cmd2(command)

        if return_code != 0:
            raise Exception(f"Process completed with non-zero exit code: {return_code}")

    except Exception as e:
        print(f"Exception occurred running {command}: {e}")
