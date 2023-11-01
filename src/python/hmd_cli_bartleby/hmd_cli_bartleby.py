import os
from pathlib import Path
from cement.utils.shell import exec_cmd2
from typing import List, Dict
from hmd_cli_tools.hmd_cli_tools import get_env_var
import json
import yaml
import urllib
from tempfile import TemporaryDirectory
import traceback

hmd_home = os.environ.get("HMD_HOME")


def get_compose(
    image_name: str,
    instance_name: str,
    transform_instance_context: Dict,
    environment: str,
    region: str,
    customer_code: str,
    deployment_id: str,
    account: str,
    autodoc: bool,
    doc_repo: str,
    doc_repo_version: str,
    input_path: str,
    output_path: str,
    pip_secret: str = None,
    document_title: str = None,
    timestamp_title: bool = False,
    confidential: bool = False,
    default_logo: str = None,
    html_default_logo: str = None,
    pdf_default_logo: str = None,
):
    env_vars = {
        "TRANSFORM_INSTANCE_CONTEXT": json.dumps(transform_instance_context),
        "HMD_ENVIRONMENT": environment,
        "HMD_REGION": region,
        "HMD_ACCOUNT": account,
        "HMD_CUSTOMER_CODE": customer_code,
        "HMD_DID": deployment_id,
        "AUTODOC": f"{autodoc}",
        "HMD_DOC_REPO_NAME": doc_repo,
        "HMD_DOC_REPO_VERSION": doc_repo_version,
        "DEFAULT_LOGO": default_logo,
        "HTML_DEFAULT_LOGO": html_default_logo,
        "PDF_DEFAULT_LOGO": pdf_default_logo,
        "HMD_DOC_COMPANY_NAME": os.environ.get("HMD_DOC_COMPANY_NAME"),
    }

    if document_title:
        env_vars["DOCUMENT_TITLE"] = document_title

    if timestamp_title:
        env_vars["NO_TIMESTAMP_TITLE"] = "true"

    if confidential and os.environ.get("HMD_BARTLEBY_CONFIDENTIALITY_STATEMENT", None):
        env_vars["CONFIDENTIALITY_STATEMENT"] = os.environ.get(
            "HMD_BARTLEBY_CONFIDENTIALITY_STATEMENT"
        )

    compose = {
        "version": "3.2",
        "services": {
            "bartleby_transform": {
                "image": image_name,
                "container_name": f"bartleby-inst_{instance_name}",
                "environment": env_vars,
                "volumes": [
                    {
                        "type": "bind",
                        "source": input_path,
                        "target": "/hmd_transform/input",
                    },
                    {
                        "type": "bind",
                        "source": output_path,
                        "target": "/hmd_transform/output",
                    },
                ],
                "secrets": [],
            }
        },
    }

    secrets = {"secrets": {}}

    if pip_secret:
        secrets["secrets"].update({"pip_url": {"file": pip_secret}})
        compose["services"]["bartleby_transform"]["secrets"].append("pip_url")
        compose["services"]["bartleby_transform"]["environment"].update(
            {"PIP_CONF": "/run/secrets/pip_url"}
        )
    compose.update(secrets)

    return compose


def transform(
    name: str,
    version: str,
    transform_instance_context: Dict,
    image_name: str,
    gather: str = None,
    autodoc: bool = False,
    confidential: bool = False,
    document_title: str = None,
    timestamp_title: bool = False,
    default_logo: str = None,
    html_default_logo: str = None,
    pdf_default_logo: str = None,
):
    if hmd_home:
        instance_name = os.environ.get("HMD_INSTANCE_NAME", name)
        deployment_id = os.environ.get("HMD_DID", "aaa")
        hmd_env = os.environ.get("HMD_ENVIRONMENT", "local")
        region = os.environ.get("HMD_REGION", "reg1")
        cust_code = os.environ.get("HMD_CUSTOMER_CODE", "hmd")
        account = os.environ.get("HMD_ACCOUNT", "")
    else:
        instance_name = get_env_var("HMD_INSTANCE_NAME")
        deployment_id = get_env_var("HMD_DID")
        hmd_env = get_env_var("HMD_ENVIRONMENT")
        region = get_env_var("HMD_REGION")
        cust_code = get_env_var("HMD_CUSTOMER_CODE")
        account = os.environ.get("HMD_ACCOUNT", "")

    repo_path = Path(os.getcwd())

    py = False
    if Path(repo_path / "src" / "python").exists():
        py = True

    input_path = repo_path

    output_path = repo_path / "target" / "bartleby"
    if not output_path.exists():
        os.makedirs(output_path)

    if not input_path.exists():
        raise Exception("No docs folder found in the current working directory.")

    if gather:
        name = gather
        for repo in gather.split(","):
            if Path(repo_path.parent / repo / "src" / "python").exists():
                py = True

    try:
        inst_config = (
            Path(os.getcwd())
            / "target"
            / "bartleby"
            / f"docker-compose-{transform_instance_context['shell']}.yaml"
        )
        if py and autodoc:
            pip_username = os.environ.get("PIP_USERNAME")
            pip_password = os.environ.get("PIP_PASSWORD")

            with TemporaryDirectory() as tempdir:
                pip_config = None
                if pip_username and pip_password:
                    pip_conf = f"""
[global]
extra-index-url = https://{pip_username}:{urllib.parse.quote(pip_password)}@hmdlabs.jfrog.io/artifactory/api/pypi/hmd_pypi/simple"""
                    pip_config = os.path.join(tempdir, "pip.conf")
                    with open(pip_config, "w") as pip:
                        pip.write(pip_conf)
                else:
                    if os.name == "nt":
                        pip_config = os.path.join(Path.home(), "pip", "pip.ini")
                    else:
                        pip_config = Path.home() / ".pip" / "pip.conf"

                print(pip_config)
                compose = get_compose(
                    image_name=image_name,
                    instance_name=instance_name,
                    transform_instance_context=transform_instance_context,
                    environment=hmd_env,
                    region=region,
                    customer_code=cust_code,
                    deployment_id=deployment_id,
                    account=account,
                    autodoc=autodoc,
                    doc_repo=name,
                    doc_repo_version=version,
                    input_path=str(input_path),
                    output_path=str(output_path),
                    pip_secret=str(pip_config),
                    confidential=confidential,
                    document_title=document_title,
                    timestamp_title=timestamp_title,
                    default_logo=default_logo,
                    html_default_logo=html_default_logo,
                    pdf_default_logo=pdf_default_logo,
                )

                with open(inst_config, "w") as conf:
                    yaml.safe_dump(compose, conf)

                command = [
                    "docker-compose",
                    "--file",
                    inst_config,
                    "up",
                    "--force-recreate",
                ]

                return_code = exec_cmd2(command)

                if return_code != 0:
                    raise Exception(
                        f"Process completed with non-zero exit code: {return_code}"
                    )

        else:
            if autodoc:
                print(
                    "Autodoc can only be used for repositories with python packages. Continuing"
                    "without autodoc enabled..."
                )
                autodoc = False
            compose = get_compose(
                image_name=image_name,
                instance_name=instance_name,
                transform_instance_context=transform_instance_context,
                environment=hmd_env,
                region=region,
                customer_code=cust_code,
                deployment_id=deployment_id,
                account=account,
                autodoc=autodoc,
                doc_repo=name,
                doc_repo_version=version,
                input_path=str(input_path),
                output_path=str(output_path),
                document_title=document_title,
                timestamp_title=timestamp_title,
                confidential=confidential,
                default_logo=default_logo,
                html_default_logo=html_default_logo,
                pdf_default_logo=pdf_default_logo,
            )

            with open(inst_config, "w") as conf:
                yaml.safe_dump(compose, conf)

            command = [
                "docker-compose",
                "--file",
                inst_config,
                "up",
                "--force-recreate",
            ]

            return_code = exec_cmd2(command)

            if return_code != 0:
                raise Exception(
                    f"Process completed with non-zero exit code: {return_code}"
                )

        rm_command = ["docker-compose", "--file", inst_config, "rm", "-f"]
        return_code = exec_cmd2(rm_command)

        if return_code != 0:
            raise Exception(
                f"Docker compose remove finished with non-zero exit code: {return_code}."
                f"Cleanup can be done manually with the following command: docker-compose --file {inst_config} rm -f"
            )

    except Exception as e:
        print(f"Exception occurred running: {e}")


def transform_puml(files: List, input_path: Path, output_path: Path, image_name: str):
    command = [
        "docker",
        "run",
        "--rm",
        "-v",
        f"{input_path}:/hmd_transform/input",
        "-v",
        f"{output_path}:/hmd_transform/output",
        "-e",
        f"PUML_FILES={','.join(files)}",
        image_name,
        "python",
        "entry_puml.py",
    ]
    print(
        f"input_path: {input_path}\n"
        f"output_path: {output_path}\n"
        f"files: {files}\n"
        f"image_name: {image_name}\n"
    )
    try:
        result = exec_cmd2(command)
    except Exception as e:
        print(f"Error executing command: {e}")

    if result != 0:
        raise Exception(f"Error generating images from puml: {traceback.format_exc()}")


def update_image(image_name: str):
    rmi_cmd = ["docker", "rmi", image_name]

    return_code = exec_cmd2(rmi_cmd)

    if return_code != 0:
        raise Exception(
            f"Removing old image completed with non-zero exit code: {return_code}"
        )

    pull_cmd = ["docker", "pull", image_name]

    return_code = exec_cmd2(pull_cmd)

    if return_code != 0:
        raise Exception(
            f"Pulling new image completed with non-zero exit code: {return_code}"
        )
