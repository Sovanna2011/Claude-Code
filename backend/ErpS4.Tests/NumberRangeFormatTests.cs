using ErpS4.Database;
using Xunit;

namespace ErpS4.Tests;

public sealed class NumberRangeFormatTests
{
    [Fact]
    public void Format_mask_fills_prefix_year_type_and_padded_number()
    {
        var formatted = NumberRangeService.Format(
            number: 123,
            prefix: "KSS",
            format: "{Prefix}-{Year}-{Type}-{Number:0000000000}",
            numberLength: 10,
            fiscalYear: 2026,
            documentType: "SA");

        Assert.Equal("KSS-2026-SA-0000000123", formatted);
    }

    [Fact]
    public void Without_a_mask_the_number_is_padded_and_prefixed()
    {
        Assert.Equal(
            "BP0000000042",
            NumberRangeService.Format(42, "BP", format: null, numberLength: 10,
                fiscalYear: 2026, documentType: null));
    }

    [Fact]
    public void Without_a_prefix_the_number_stands_alone()
    {
        Assert.Equal(
            "0000000042",
            NumberRangeService.Format(42, prefix: null, format: null, numberLength: 10,
                fiscalYear: 2026, documentType: null));
    }
}
