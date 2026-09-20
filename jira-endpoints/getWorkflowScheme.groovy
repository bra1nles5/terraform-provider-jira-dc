import com.atlassian.jira.component.ComponentAccessor
import com.atlassian.jira.project.ProjectManager
import com.atlassian.jira.workflow.WorkflowSchemeManager
import com.onresolve.scriptrunner.runner.rest.common.CustomEndpointDelegate
import groovy.transform.BaseScript
import javax.ws.rs.core.Response
import groovy.json.JsonSlurper
import groovy.json.JsonOutput

@BaseScript CustomEndpointDelegate delegate

getWorkflowScheme(
    httpMethod: "GET",
    groups: ["jira-administrators"]
) { params ->

    def responseBody = [:]
    def status = 200

    try {
        def projectKey = params.getFirst("projectKey")
        if (!projectKey) {
            return Response.status(400)
                .entity(JsonOutput.toJson([error: "projectKey is required"]))
                .build()
        }

        def projectManager = ComponentAccessor.getProjectManager()
        def workflowManager = ComponentAccessor.getComponent(WorkflowSchemeManager)

        def project = projectManager.getProjectObjByKeyIgnoreCase(projectKey)
        if (!project) {
            return Response.status(404)
                .entity(JsonOutput.toJson([error: "Project '${projectKey}' not found"]))
                .build()
        }

        def scheme = workflowManager.getWorkflowSchemeObj(project)
        if (!scheme) {
            return Response.status(404)
                .entity(JsonOutput.toJson([error: "Workflow scheme not found"]))
                .build()
        }

        responseBody = [
            success    : true,
            projectKey : projectKey,
            schemeId   : scheme.id as Long,
            schemeName : scheme.name
        ]

    } catch (Exception e) {
        log.error("Error in getWorkflowScheme: ${e.message}", e)
        status = 500
        responseBody = [error: e.message ?: "Internal Server Error"]
    }

    return Response.status(status)
        .entity(JsonOutput.toJson(responseBody))
        .build()
}