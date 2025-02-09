package pages

import (
	"fmt"
	"net/http"

	"github.com/supporttools/KubeCertWatch/pkg/checks"
)

// IngressStatusPage renders a table showing the status of Ingress SSL checks.
func IngressStatusPage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	statuses := checks.GetIngressStatuses()

	fmt.Fprint(w, `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Ingress Status</title>
			<style>
				table {
					border-collapse: collapse;
					width: 100%;
				}
				th, td {
					border: 1px solid #ddd;
					padding: 8px;
				}
				th {
					cursor: pointer;
					background-color: #f2f2f2;
				}
				.filter-input {
					margin-bottom: 10px;
					padding: 5px;
					width: 300px;
				}
			</style>
			<script>
				function filterTable() {
					let filter = document.getElementById("filterInput").value.toUpperCase();
					let table = document.getElementById("statusTable");
					let rows = table.getElementsByTagName("tr");

					for (let i = 1; i < rows.length; i++) {
						let cells = rows[i].getElementsByTagName("td");
						let match = false;
						for (let j = 0; j < cells.length; j++) {
							if (cells[j].innerText.toUpperCase().indexOf(filter) > -1) {
								match = true;
								break;
							}
						}
						rows[i].style.display = match ? "" : "none";
					}
				}
			</script>
		</head>
		<body>
			<h1>Ingress SSL Status</h1>
			<input type="text" id="filterInput" class="filter-input" onkeyup="filterTable()" placeholder="Search for ingress...">
			<table id="statusTable">
				<tr>
					<th>Namespace</th>
					<th>Ingress</th>
					<th>Internal SSL</th>
					<th>External SSL</th>
				</tr>
	`)

	for _, status := range statuses {
		fmt.Fprintf(w, `
			<tr>
				<td>%s</td>
				<td>%s</td>
				<td>%s</td>
				<td>%s</td>
			</tr>
		`, status.Namespace, status.IngressName, status.InternalStatus, status.ExternalStatus)
	}

	fmt.Fprint(w, `
			</table>
		</body>
		</html>
	`)
}
