using System.Globalization;
using System.Text;
using System.Text.Json;
using ClosedXML.Excel;
using QuestPDF.Fluent;
using QuestPDF.Helpers;
using QuestPDF.Infrastructure;
using SugarcanePlanning.Application.Interfaces;
using SugarcanePlanning.Contracts.Reports;

namespace SugarcanePlanning.Infrastructure.Reporting;

/// <summary>Writes a report result as a landscape A4 PDF (section 20).</summary>
public class PdfReportExporter : IReportExporter
{
    public ExportFormat Format => ExportFormat.Pdf;
    public string ContentType => "application/pdf";
    public string FileExtension => "pdf";

    static PdfReportExporter() => QuestPDF.Settings.License = LicenseType.Community;

    public byte[] Export(ReportResultDto report)
    {
        var document = Document.Create(container =>
        {
            container.Page(page =>
            {
                page.Size(PageSizes.A4.Landscape());
                page.Margin(18);
                page.DefaultTextStyle(t => t.FontSize(8).FontFamily(Fonts.Calibri));

                page.Header().Column(header =>
                {
                    header.Item().Text(report.Title).FontSize(14).SemiBold();
                    if (!string.IsNullOrWhiteSpace(report.Subtitle))
                        header.Item().Text(report.Subtitle).FontSize(9).FontColor(Colors.Grey.Darken1);
                    var filters = report.Filters.Count == 0
                        ? "no filters"
                        : string.Join(" · ", report.Filters.Select(f => $"{f.Key}: {f.Value}"));
                    header.Item().Text($"Filters: {filters}").FontSize(7).FontColor(Colors.Grey.Darken1);
                    header.Item().PaddingBottom(4).Text(
                            $"Generated {report.GeneratedAtUtc:yyyy-MM-dd HH:mm} UTC by {report.GeneratedBy}")
                        .FontSize(7).FontColor(Colors.Grey.Darken1);
                });

                page.Content().Table(table =>
                {
                    table.ColumnsDefinition(columns =>
                    {
                        foreach (var _ in report.Columns) columns.RelativeColumn();
                    });

                    table.Header(header =>
                    {
                        foreach (var column in report.Columns)
                            header.Cell().Background(Colors.Grey.Lighten2).Padding(3)
                                .Text(column.Header).SemiBold().FontSize(8);
                    });

                    foreach (var row in report.Rows)
                    {
                        foreach (var column in report.Columns)
                        {
                            var cell = table.Cell().BorderBottom(0.5f).BorderColor(Colors.Grey.Lighten2).Padding(3);
                            var text = row.Values.GetValueOrDefault(column.Key) ?? string.Empty;
                            if (column.IsNumeric) cell.AlignRight().Text(text).FontSize(8);
                            else cell.Text(text).FontSize(8);
                        }
                    }

                    if (report.Totals.Count > 0)
                    {
                        foreach (var column in report.Columns)
                        {
                            var cell = table.Cell().Background(Colors.Grey.Lighten3).Padding(3);
                            if (report.Totals.TryGetValue(column.Key, out var total))
                                cell.AlignRight().Text(total.ToString("N2", CultureInfo.InvariantCulture)).SemiBold();
                            else if (column.Key == report.Columns[0].Key)
                                cell.Text("Total").SemiBold();
                            else cell.Text(string.Empty);
                        }
                    }
                });

                page.Footer().AlignCenter().Text(t =>
                {
                    t.Span("Page ").FontSize(7);
                    t.CurrentPageNumber().FontSize(7);
                    t.Span(" / ").FontSize(7);
                    t.TotalPages().FontSize(7);
                });
            });
        });

        return document.GeneratePdf();
    }
}

/// <summary>Writes a report result as an .xlsx workbook with an auto-filtered, frozen header (section 20).</summary>
public class ExcelReportExporter : IReportExporter
{
    public ExportFormat Format => ExportFormat.Excel;
    public string ContentType => "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet";
    public string FileExtension => "xlsx";

    public byte[] Export(ReportResultDto report)
    {
        using var workbook = new XLWorkbook();
        var sheetName = report.Report.ToString();
        var sheet = workbook.Worksheets.Add(sheetName[..Math.Min(sheetName.Length, 31)]);

        var row = 1;
        sheet.Cell(row, 1).Value = report.Title;
        sheet.Cell(row, 1).Style.Font.Bold = true;
        sheet.Cell(row, 1).Style.Font.FontSize = 14;
        row++;

        if (!string.IsNullOrWhiteSpace(report.Subtitle))
        {
            sheet.Cell(row, 1).Value = report.Subtitle;
            row++;
        }

        sheet.Cell(row, 1).Value = $"Generated {report.GeneratedAtUtc:yyyy-MM-dd HH:mm} UTC by {report.GeneratedBy}";
        row++;

        if (report.Filters.Count > 0)
        {
            sheet.Cell(row, 1).Value = "Filters: " + string.Join(" · ", report.Filters.Select(f => $"{f.Key}={f.Value}"));
            row++;
        }
        row++;

        var headerRow = row;
        for (var c = 0; c < report.Columns.Count; c++)
        {
            var cell = sheet.Cell(headerRow, c + 1);
            cell.Value = report.Columns[c].Header;
            cell.Style.Font.Bold = true;
            cell.Style.Fill.BackgroundColor = XLColor.LightGray;
        }
        row++;

        foreach (var dataRow in report.Rows)
        {
            for (var c = 0; c < report.Columns.Count; c++)
            {
                var column = report.Columns[c];
                var text = dataRow.Values.GetValueOrDefault(column.Key);
                var cell = sheet.Cell(row, c + 1);

                if (column.IsNumeric && decimal.TryParse(text, NumberStyles.Any, CultureInfo.InvariantCulture, out var number))
                {
                    cell.Value = number;
                    cell.Style.NumberFormat.Format = column.DataType == "percent" ? "#,##0.00\"%\"" : "#,##0.0000";
                }
                else if (column.DataType == "date" && DateTime.TryParse(text, CultureInfo.InvariantCulture,
                             DateTimeStyles.None, out var date))
                {
                    cell.Value = date;
                    cell.Style.NumberFormat.Format = "yyyy-mm-dd";
                }
                else
                {
                    cell.Value = text ?? string.Empty;
                }

                if (dataRow.IsSubtotal) cell.Style.Font.Bold = true;
            }
            row++;
        }

        if (report.Totals.Count > 0)
        {
            sheet.Cell(row, 1).Value = "Total";
            sheet.Cell(row, 1).Style.Font.Bold = true;
            for (var c = 0; c < report.Columns.Count; c++)
            {
                if (!report.Totals.TryGetValue(report.Columns[c].Key, out var total)) continue;
                var cell = sheet.Cell(row, c + 1);
                cell.Value = total;
                cell.Style.Font.Bold = true;
                cell.Style.NumberFormat.Format = "#,##0.0000";
            }
        }

        if (report.Rows.Count > 0)
            sheet.Range(headerRow, 1, headerRow + report.Rows.Count, report.Columns.Count).SetAutoFilter();
        sheet.SheetView.FreezeRows(headerRow);
        sheet.Columns().AdjustToContents(8d, 45d);

        using var stream = new MemoryStream();
        workbook.SaveAs(stream);
        return stream.ToArray();
    }
}

/// <summary>Raw JSON, used by the print-preview screen and by integrations.</summary>
public class JsonReportExporter : IReportExporter
{
    private static readonly JsonSerializerOptions Options = new() { WriteIndented = true };

    public ExportFormat Format => ExportFormat.Json;
    public string ContentType => "application/json";
    public string FileExtension => "json";

    public byte[] Export(ReportResultDto report)
        => Encoding.UTF8.GetBytes(JsonSerializer.Serialize(report, Options));
}
