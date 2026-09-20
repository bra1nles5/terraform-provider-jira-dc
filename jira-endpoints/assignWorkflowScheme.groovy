import com.atlassian.jira.component.ComponentAccessor
import com.atlassian.jira.project.Project
import com.atlassian.jira.scheme.Scheme
import com.atlassian.jira.workflow.WorkflowSchemeManager
import com.onresolve.scriptrunner.runner.rest.common.CustomEndpointDelegate
import groovy.json.JsonBuilder
import groovy.json.JsonSlurper
import groovy.transform.BaseScript
import javax.ws.rs.core.MultivaluedMap
import javax.ws.rs.core.Response

@BaseScript CustomEndpointDelegate delegate

assignWorkflowScheme(
    httpMethod: "PUT",
    groups: ["jira-administrators"]
) { MultivaluedMap queryParams, String body ->

    def responseBody = [:]
    def status = 200

    try {
        if (!body?.trim()) {
            return Response.status(400).entity(new JsonBuilder([error: "JSON body required"]).toString()).build()
        }

        Map json = new JsonSlurper().parseText(body) as Map

        def projectKey  = json.projectKey as String
        def schemeId    = json.schemeId as Long
        def updateDraft = json.updateDraft as Boolean ?: true

        if (!projectKey || schemeId == null) {
            return Response.status(400).entity(new JsonBuilder([error: "projectKey and schemeId are required"]).toString()).build()
        }

        def projectManager = ComponentAccessor.projectManager
        def schemeManager  = ComponentAccessor.getComponent(WorkflowSchemeManager)

        Project project = projectManager.getProjectObjByKeyIgnoreCase(projectKey)
        if (!project) {
            return Response.status(404).entity(new JsonBuilder([error: "Project '${projectKey}' not found"]).toString()).build()
        }

        Scheme scheme = schemeManager.getSchemeObject(schemeId)

        if (!scheme) {
            return Response.status(404).entity(new JsonBuilder([error: "WorkflowScheme ID ${schemeId} not found"]).toString()).build()
        }

        schemeManager.removeSchemesFromProject(project)

        schemeManager.addSchemeToProject(project, scheme)

        responseBody = [
            success    : true,
            projectKey : projectKey,
            schemeId   : schemeId,
            schemeName : scheme.name,
            message    : "The workflow schema has been successfully assigned to the project. If the project contains tasks, the migration will start automatically (this may take some time)."
        ]

    } catch (Exception e) {
        log.error("Error in assignWorkflowScheme: ${e.message}", e)
        status = 500
        responseBody = [error: e.message ?: "Internal Server Error"]
    }

    return Response.status(status).entity(new JsonBuilder(responseBody).toString()).build()
}