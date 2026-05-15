namespace MyApp.Domain.ValueObjects
{
    /// <summary>
    /// Email value object with validation
    /// </summary>
    public class Email
    {
        public string Address { get; }

        public Email(string address)
        {
            if (string.IsNullOrWhiteSpace(address))
            {
                throw new ArgumentException("Email cannot be empty");
            }
            Address = address;
        }
    }
}
