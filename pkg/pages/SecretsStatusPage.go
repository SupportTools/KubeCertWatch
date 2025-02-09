package pages

import (
	"fmt"
	"net/http"

	"github.com/supporttools/KubeCertWatch/pkg/checks"
)

// SecretsStatusPage provides a simple HTML page displaying the status of TLS secrets
func SecretsStatusPage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	statuses := checks.GetSecretStatuses()

	fmt.Fprint(w, `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Secrets Status</title>
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

				function sortTable(columnIndex) {
					let table = document.getElementById("statusTable");
					let rows = Array.from(table.rows).slice(1);
					let ascending = table.getAttribute("data-sort-order") !== "asc";
					table.setAttribute("data-sort-order", ascending ? "asc" : "desc");

					rows.sort((a, b) => {
						let cellA = a.cells[columnIndex].innerText.toUpperCase();
						let cellB = b.cells[columnIndex].innerText.toUpperCase();
						if (!isNaN(cellA) && !isNaN(cellB)) {
							return ascending ? cellA - cellB : cellB - cellA;
						}
						return ascending
							? cellA.localeCompare(cellB)
							: cellB.localeCompare(cellA);
					});

					rows.forEach(row => table.appendChild(row));
				}
			</script>
		</head>
		<body>
			<h1>Secrets Status</h1>
			<input type="text" id="filterInput" class="filter-input" onkeyup="filterTable()" placeholder="Search for secrets...">
			<table id="statusTable" data-sort-order="asc">
				<tr>
					<th onclick="sortTable(0)">Namespace</th>
					<th onclick="sortTable(1)">Secret Name</th>
					<th onclick="sortTable(2)">Expiration Date</th>
					<th onclick="sortTable(3)">Days Until</th>
					<th onclick="sortTable(4)">Status</th>
				</tr>
	`)

	for _, status := range statuses {
		fmt.Fprintf(w, `
			<tr>
				<td>%s</td>
				<td>%s</td>
				<td>%s</td>
				<td>%d</td>
				<td>%s</td>
			</tr>
		`, status.Namespace, status.SecretName, status.ExpirationDate, status.DaysUntil, status.Status)
	}

	fmt.Fprint(w, `
			</table>
		</body>
		</html>
	`)
}
