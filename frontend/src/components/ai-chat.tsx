import { useState } from "react";
import { toast } from "sonner";
import axios from "axios";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";

type AIResponse = {
  summary: string;
  findings: string[];
  recommendations: string[];
};

type Message = {
  role: "user" | "assistant";
  content: string | AIResponse;
};

export function AIChat() {
  const [repository, setRepository] = useState("");
  const [question, setQuestion] = useState("");
  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(false);

  async function sendMessage() {
    if (!repository.trim() || !question.trim() || loading) return;

    const currentQuestion = question.trim();

    setMessages((prev) => [
      ...prev,
      { role: "user", content: currentQuestion },
    ]);
    setQuestion("");
    setLoading(true);

    try {
      const { data } = await axios.post<AIResponse>("/api/chat", {
        repository: repository.trim(),
        question: currentQuestion,
      });

      setMessages((prev) => [...prev, { role: "assistant", content: data }]);
    } catch (error) {
      console.error("AI chat error:", error);
      toast.error("Could not connect to the AI service.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="mx-auto flex h-full w-full max-w-3xl flex-1 flex-col gap-6">
      <div>
        <h1 className="font-serif text-3xl font-semibold tracking-tight">
          Chat with AI
        </h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Ask questions about your GitHub repository.
        </p>
      </div>

      <div className="space-y-2">
        <label htmlFor="repository" className="text-sm font-medium">
          GitHub Repository
        </label>
        <Input
          id="repository"
          placeholder="owner/repository"
          value={repository}
          onChange={(e) => setRepository(e.target.value)}
          disabled={loading}
        />
      </div>

      <div className="flex min-h-48 flex-1 flex-col gap-4 overflow-y-auto">
        {messages.length === 0 && (
          <div className="flex flex-1 items-center justify-center text-sm text-muted-foreground">
            Ask your first question to get started.
          </div>
        )}

        {messages.map((message, index) => (
          <Card
            key={index}
            className={
              message.role === "user"
                ? "ml-auto max-w-[85%] bg-muted"
                : "mr-auto w-full"
            }
          >
            <CardContent className="space-y-3 text-sm">
              {typeof message.content === "string" ? (
                <p>{message.content}</p>
              ) : (
                <>
                  <p>{message.content.summary}</p>

                  {message.content.findings.length > 0 && (
                    <div>
                      <h3 className="mb-2 font-semibold">Findings</h3>
                      <ul className="list-disc space-y-1 pl-5">
                        {message.content.findings.map((finding, i) => (
                          <li key={i}>{finding}</li>
                        ))}
                      </ul>
                    </div>
                  )}

                  {message.content.recommendations.length > 0 && (
                    <div>
                      <h3 className="mb-2 font-semibold">Recommendations</h3>
                      <ul className="list-disc space-y-1 pl-5">
                        {message.content.recommendations.map((item, i) => (
                          <li key={i}>{item}</li>
                        ))}
                      </ul>
                    </div>
                  )}
                </>
              )}
            </CardContent>
          </Card>
        ))}

        {loading && (
          <p className="text-sm text-muted-foreground">
            Analyzing repository...
          </p>
        )}
      </div>

      <div className="space-y-3 border-t pt-4">
        <Textarea
          placeholder="What would you like to know about this project?"
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          disabled={loading}
          rows={3}
        />

        <div className="flex justify-end">
          <Button
            onClick={sendMessage}
            disabled={!repository.trim() || !question.trim() || loading}
          >
            {loading ? "Analyzing..." : "Send message"}
          </Button>
        </div>
      </div>
    </div>
  );
}
