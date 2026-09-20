import com.atlassian.jira.component.ComponentAccessor
import com.atlassian.jira.config.properties.APKeys
import com.atlassian.jira.workflow.WorkflowSchemeManager
import com.onresolve.scriptrunner.runner.rest.common.CustomEndpointDelegate
import groovy.transform.BaseScript
import javax.ws.rs.core.Response
import groovy.json.JsonOutput

@BaseScript CustomEndpointDelegate delegate

listWorkflowSchemes(
    httpMethod: "GET",
    groups: ["jira-administrators"]
) { params ->

    def responseBody = [:]
    def status = 200

    try {
        def workflowManager = ComponentAccessor.getComponent(WorkflowSchemeManager)
        def baseUrl = ComponentAccessor.getApplicationProperties().getString(APKeys.JIRA_BASEURL)

        def schemes = workflowManager.getSchemeObjects().collect { scheme ->
            [
                self       : "${baseUrl}/rest/api/2/workflowscheme/${scheme.id}".toString(),
                id         : scheme.id as Long,
                name       : scheme.name,
                description: scheme.description ?: ""
            ]
        }

        responseBody = [schemes: schemes]

    } catch (Exception e) {
        log.error("Error in listWorkflowSchemes: ${e.message}", e)
        status = 500
        responseBody = [error: e.message ?: "Internal Server Error"]
    }

    return Response.status(status)
        .entity(JsonOutput.toJson(responseBody))
        .build()
}
