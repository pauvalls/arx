using System;
using MyApp.Application.Services;

namespace MyApp
{
    class Program
    {
        static void Main(string[] args)
        {
            Console.WriteLine("MyApp - C# Architecture Test Project");
            Console.WriteLine("====================================");
            
            // This is just a test fixture for arx detector testing
            var userId = Guid.NewGuid();
            Console.WriteLine($"Test user ID: {userId}");
        }
    }
}
