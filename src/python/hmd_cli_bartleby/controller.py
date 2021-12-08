import json
from importlib.metadata import version
from cement import Controller, ex
from hmd_cli_tools.hmd_cli_tools import set_pargs_value


VERSION_BANNER = """
hmd bartleby version: {}
"""

run_arguments = [
    (
        [],
        {
            "action": "store",
            "dest": "shell",
            "help": "The command to pass to the bartleby transform instance",
            "default": "default",
        },
    )
]


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
        )

    def _default(self):
        """Default action if no sub-command is passed."""

        self.app.args.print_help()

    @ex(help="Run the bartleby transform.", arguments=run_arguments, aliases=["tf"])
    def transform(self):
        args = {}
        name = self.app.pargs.repo_name
        version = self.app.pargs.repo_version

        for _, arg_def in run_arguments:
            set_pargs_value(self.app.pargs, arg_def["dest"], arg_def.get("default"))

        image_name = "ghcr.io/hmdlabs/hmd-tf-bartleby:test"

        transform_instance_context = json.dumps({"shell": f"{self.app.pargs.shell}"})

        args.update(
            {
                "name": name,
                "version": version,
                "transform_instance_context": transform_instance_context,
                "image_name": image_name,
            }
        )

        from .hmd_cli_bartleby import transform

        transform(**args)
