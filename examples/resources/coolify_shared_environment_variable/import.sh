# The import ID format depends on scope:
terraform import coolify_shared_environment_variable.team_wide team/<key>
terraform import coolify_shared_environment_variable.per_project project/<project_uuid>/<key>
terraform import coolify_shared_environment_variable.per_env environment/<project_uuid>/<environment>/<key>
terraform import coolify_shared_environment_variable.per_server server/<server_uuid>/<key>
