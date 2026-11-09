import { useState } from "react";
import { toast } from "sonner";
import axios from "axios";
import { useSearchParams } from "react-router";
import { readDefaultRepository } from "@/lib/preferences";

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
  const [params] = useSearchParams();
  const [repository, setRepository] = useState(() => params.get("repository") ?? readDefaultRepository());
  const [question, setQuestion] = useState(() => params.get("question") ?? "");
  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(false);

  async function sendMessage() {
    if (!repository.trim() || !question.trim() || loading) return;

    const currentQuestion = question.trim();

    setLoading(true);

    try {
      const { data } = await axios.post<AIResponse>(
        "/api/chat",
        { repository: repository.trim(), question: currentQuestion },
        { timeout: 40_000 },
      );

      setMessages((prev) => [
        ...prev,
        { role: "user", content: currentQuestion },
        { role: "assistant", content: data },
      ]);
      setQuestion("");
    } catch (error) {
      let message = "Could not reach the AI service. Check your connection and try again.";
      if (axios.isAxiosError(error) && error.response) {
        switch (error.response.status) {
          case 400:
            message = "Enter a repository name and a question, then try again.";
            break;
          case 413:
            message = "Your request is too long. Shorten the repository name or question and try again.";
            break;
          case 502:
            message = "The AI service could not generate a response. Please try again.";
            break;
          default:
            message = "The server could not complete your request. Please try again.";
        }
      }
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="mx-auto flex h-full w-full max-w-3xl flex-1 flex-col gap-6">
      <div>
        <h1 className="text-4xl leading-none font-normal tracking-[-0.04em] md:text-5xl">
          Chat with AI
        </h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Get software-project guidance using a repository name and your question.
          DevPulse does not retrieve or inspect GitHub data.
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

      <div
        role="log"
        aria-label="Chat messages"
        aria-busy={loading}
        className="flex min-h-48 flex-1 flex-col gap-4 overflow-y-auto"
      >
        {messages.length === 0 && !loading && (
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
            <CardContent className="space-y-3 text-sm wrap-anywhere">
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
          <p role="status" className="text-sm text-muted-foreground">
            Generating a response...
          </p>
        )}
      </div>

      <div className="space-y-3 border-t pt-4">
        <label htmlFor="question" className="text-sm font-medium">
          Your question
        </label>
        <Textarea
          id="question"
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
            {loading ? "Generating..." : "Send message"}
          </Button>
        </div>
      </div>
    </div>
  );
}
