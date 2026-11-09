import { useState } from "react";
import { toast } from "sonner";
import axios from "axios";
import { useSearchParams } from "react-router";
import { isRepositoryName, readDefaultRepository } from "@/lib/preferences";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import type { AuthState } from "@/hooks/use-session";
import { RepositoryPicker } from "@/components/repository-picker";
import { AnalysisResult } from "@/components/analysis-result";
import { requestError, type AIResponse } from "@/lib/analysis";
import { Textarea } from "@/components/ui/textarea";

type Message = {
  role: "user" | "assistant";
  content: string | AIResponse;
};

export function AIChat({ auth }: { auth: AuthState }) {
  const [params] = useSearchParams();
  const [repository, setRepository] = useState(() => params.get("repository") ?? readDefaultRepository());
  const [question, setQuestion] = useState(() => params.get("question") ?? "");
  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(false);

  async function sendMessage() {
    if (!repository.trim() || !question.trim() || loading) return;

    if (!isRepositoryName(repository.trim())) {
      toast.error("Use the format owner/repository.");
      return;
    }
    const currentQuestion = question.trim();

    setLoading(true);

    try {
      const { data } = await axios.post<AIResponse>(
        "/api/chat",
        { repository: repository.trim(), question: currentQuestion },
        { timeout: 75_000 },
      );

      setMessages((prev) => [
        ...prev,
        { role: "user", content: currentQuestion },
        { role: "assistant", content: data },
      ]);
      setQuestion("");
    } catch (error) {
      toast.error(requestError(error));
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
          Ask about your project using its GitHub file names and README as context.
        </p>
      </div>

      <RepositoryPicker user={auth.user} value={repository} onChange={(value) => { setRepository(value); setMessages([]); }} disabled={loading} />

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
                <AnalysisResult response={message.content} />
              )}
            </CardContent>
          </Card>
        ))}

        {loading && (
          <p role="status" className="text-sm text-muted-foreground">
            Reading repository context and generating a response...
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
