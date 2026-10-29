import QtQuick
import Quickshell
import Quickshell.Io

Item {
    id: root
    property var bar
    property var settings
    property string moduleName: "sabrodigan.todo"
    property string tooltipText: "Today’s Todos v1.5.0\nby Stephen Brodigan"
    readonly property bool vertical: bar ? bar.vertical : false
    readonly property bool tooltipHovered: hoverArea.containsMouse
    readonly property string executable: decodeURIComponent(Qt.resolvedUrl("bin/todo").toString().replace(/^file:\/\//, ""))
    property int total: 0
    property int today: 0
    property int overdue: 0
    property bool loaded: false
    property bool failed: false
    property bool refreshPending: false
    implicitWidth: vertical ? (bar ? bar.barSize : 26) : label.implicitWidth + 16
    implicitHeight: vertical ? label.implicitHeight + 10 : (bar ? bar.barSize : 26)

    function refresh() {
        if (query.running) refreshPending = true
        else query.running = true
    }
    function triggerPress(button) {
        if (bar) bar.hideTooltip(root)
        Quickshell.execDetached(["omarchy", "launch", "terminal", executable, "interactive"])
    }
    Process {
        id: query
        command: [root.executable, "status"]
        stdout: StdioCollector {
            onStreamFinished: {
                try {
                    const result = JSON.parse(text)
                    if (typeof result.total !== "number") throw new Error("Update todo CLI")
                    root.total = result.total
                    root.today = result.today
                    root.overdue = result.overdue
                    root.loaded = true
                    root.failed = false
                } catch (error) { root.failed = true }
            }
        }
        onExited: function(code) {
            if (code !== 0) root.failed = true
            if (root.refreshPending) {
                root.refreshPending = false
                root.refresh()
            }
        }
    }
    IpcHandler {
        target: "sabrodigan.todo"
        function refresh(): void { root.refresh() }
    }
    Timer {
        interval: 5000
        running: true
        repeat: true
        triggeredOnStart: true
        onTriggered: root.refresh()
    }
    MouseArea {
        id: hoverArea
        anchors.fill: parent
        acceptedButtons: Qt.NoButton
        hoverEnabled: true
        onEntered: {
            if (root.bar) root.bar.showTooltip(root, root.tooltipText)
        }
        onExited: {
            if (root.bar) root.bar.hideTooltip(root)
        }
    }
    Text {
        id: label
        anchors.centerIn: parent
        textFormat: Text.StyledText
        text: {
            if (root.failed) return "☑ !"
            if (!root.loaded) return "☑ …"
            const separator = root.vertical ? "<br>" : " · "
            return "☑ " + root.total + separator
                + '<font color="#22c55e">' + root.today + '</font>' + separator
                + '<font color="#ef4444">' + root.overdue + '</font>'
        }
        color: bar ? bar.foreground : "white"
        font.family: bar ? bar.fontFamily : "monospace"
        font.pixelSize: 12
    }
}
