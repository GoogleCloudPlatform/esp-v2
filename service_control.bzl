def servicecontrol_client_repositories(bind = True):

    native.git_repository(
        name = "servicecontrol_client_git",
        commit = "8189638f8e2c410010befe5dbd81e267a15e3e17",
        remote = "https://github.com/cloudendpoints/service-control-client-cxx.git",
    )

    if bind:
        native.bind(
            name = "servicecontrol_client",
            actual = "@servicecontrol_client_git//:service_control_client_lib",
        )
        native.bind(
            name = "quotacontrol",
            actual = "@servicecontrol_client_git//proto:quotacontrol",
        )
        native.bind(
            name = "quotacontrol_genproto",
            actual = "@servicecontrol_client_git//proto:quotacontrol_genproto",
        )