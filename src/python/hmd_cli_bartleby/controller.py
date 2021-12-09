import json
from importlib.metadata import version
from cement import Controller, ex
from hmd_cli_tools.hmd_cli_tools import set_pargs_value


VERSION_BANNER = """
hmd bartleby version: {}
"""


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
                [],
                {
                    "action": "store",
                    "dest": "shell",
                    "help": "The command to pass to the bartleby transform instance",
                    "default": "default",
                },
            ),
        )

    def _default(self):
        """Default action if no sub-command is passed."""

        args = {}
        name = self.app.pargs.repo_name
        repo_version = self.app.pargs.repo_version
        image_name = f"ghcr.io/hmdlabs/hmd-tf-bartleby"
        shell = self.app.pargs.shell

        if len(shell.split(",")) > 1:
            for cmd in shell.split(","):
                transform_instance_context = json.dumps({"shell": f"{cmd.strip()}"})
                args.update(
                    {
                        "name": name,
                        "version": repo_version,
                        "transform_instance_context": transform_instance_context,
                        "image_name": image_name,
                    }
                )

                from .hmd_cli_bartleby import transform

                transform(**args)
        else:
            transform_instance_context = json.dumps({"shell": f"{shell}"})

            args.update(
                {
                    "name": name,
                    "version": repo_version,
                    "transform_instance_context": transform_instance_context,
                    "image_name": image_name,
                }
            )

            from .hmd_cli_bartleby import transform

            transform(**args)
